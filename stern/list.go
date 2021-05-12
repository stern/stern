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
	"sync"

	"github.com/pkg/errors"
	"github.com/stern/stern/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// List returns a list of all
func List(ctx context.Context, config *Config) (map[string]string, error) {
	clientConfig := kubernetes.NewClientConfig(config.KubeConfig, config.ContextName)
	cc, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := corev1client.NewForConfig(cc)
	if err != nil {
		return nil, err
	}

	var namespaces []string
	// A specific namespace is ignored if all-namespaces is provided.
	if config.AllNamespaces {
		namespaces = []string{""}
	} else {
		namespaces = config.Namespaces
		if len(namespaces) == 0 {
			n, _, err := clientConfig.Namespace()
			if err != nil {
				return nil, errors.Wrap(err, "unable to get default namespace")
			}
			namespaces = []string{n}
		}
	}

	labels := make(map[string]string)
	options := metav1.ListOptions{}

	wg := sync.WaitGroup{}

	wg.Add(len(namespaces))

	// Concurrently iterate through provided namespaces.
	for _, n := range namespaces {
		go func(n string) {
			defer wg.Done()

			pods, err := clientset.Pods(n).List(ctx, options)
			if err != nil {
				return
			}

			match := "app.kubernetes.io/instance"
			// Iterate through pods in namespace, looking for matching labels.
			for _, pod := range pods.Items {
				key := pod.Labels[match]

				if key == "" {
					continue
				}

				labels[key] = match
			}
		}(n)
	}

	wg.Wait()

	return labels, nil
}
