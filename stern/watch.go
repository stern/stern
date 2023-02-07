//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package stern

import (
	"context"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

// Watch starts listening to Kubernetes events and emits modified
// containers/pods. The result is targets added.
func WatchTargets(ctx context.Context, i v1.PodInterface, labelSelector labels.Selector, fieldSelector fields.Selector, filter *targetFilter) (chan *Target, error) {
	// RetryWatcher will make sure that in case the underlying watcher is
	// closed (e.g. due to API timeout or etcd timeout) it will get restarted
	// from the last point without the consumer even knowing about it.
	watcher, err := watchtools.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return i.Watch(ctx, metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a watcher")
	}

	added := make(chan *Target)
	go func() {
		for {
			select {
			case e := <-watcher.ResultChan():
				if e.Object == nil {
					// Closed because of error
					close(added)
					return
				}

				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					continue
				}

				switch e.Type {
				case watch.Added, watch.Modified:
					filter.visit(pod, func(t *Target) {
						added <- t
					})
				case watch.Deleted:
					filter.forget(string(pod.UID))
				}
			case <-ctx.Done():
				watcher.Stop()
				close(added)
				return
			}
		}
	}()

	return added, nil
}
