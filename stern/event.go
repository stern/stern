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
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

// eventMatcher reports whether an event should be shown.
type eventMatcher func(*corev1.Event) bool

// WatchEvents watches Kubernetes Events in a namespace and writes the ones
// accepted by match to out. It runs until ctx is cancelled or the watch is
// closed.
func WatchEvents(ctx context.Context, i v1.EventInterface, match eventMatcher, out io.Writer) error {
	// RetryWatcher restarts the underlying watch if it drops, the same way
	// WatchTargets watches pods.
	watcher, err := watchtools.NewRetryWatcherWithContext(ctx, "1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return i.Watch(ctx, metav1.ListOptions{})
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create an event watcher")
	}
	defer watcher.Stop()

	return consumeEvents(ctx, watcher.ResultChan(), match, out)
}

// consumeEvents reads watch events off ch and writes the matched ones to out.
// It returns when ch is closed or ctx is cancelled.
func consumeEvents(ctx context.Context, ch <-chan watch.Event, match eventMatcher, out io.Writer) error {
	for {
		select {
		case e, ok := <-ch:
			if !ok || e.Object == nil {
				return nil
			}
			// We only care about newly observed events, not bookkeeping
			// updates the server makes to an existing one (count bumps etc.).
			if e.Type != watch.Added {
				continue
			}
			event, ok := e.Object.(*corev1.Event)
			if !ok {
				continue
			}
			if match != nil && !match(event) {
				continue
			}
			fmt.Fprint(out, formatEvent(event))
		case <-ctx.Done():
			return nil
		}
	}
}

// formatEvent renders an event on a single line with a distinct "event" marker
// so it stands out from log lines. Warning events are red, everything else is
// yellow.
func formatEvent(e *corev1.Event) string {
	marker := color.New(color.FgHiYellow, color.Bold)
	if e.Type == corev1.EventTypeWarning {
		marker = color.New(color.FgHiRed, color.Bold)
	}

	obj := e.InvolvedObject.Name
	if e.InvolvedObject.Kind != "" {
		obj = e.InvolvedObject.Kind + "/" + obj
	}

	msg := strings.TrimSpace(e.Message)

	return fmt.Sprintf("%s %s %s %s: %s\n",
		marker.Sprint("event"),
		marker.Sprint(e.Type),
		obj,
		e.Reason,
		msg,
	)
}
