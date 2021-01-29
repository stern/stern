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
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/stern/stern/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

var tails = make(map[string]*Tail)
var tailLock sync.RWMutex

func getTail(targetID string) (*Tail, bool) {
	tailLock.RLock()
	defer tailLock.RUnlock()
	tail, ok := tails[targetID]
	return tail, ok
}

func setTail(targetID string, tail *Tail) {
	tailLock.Lock()
	defer tailLock.Unlock()
	tails[targetID] = tail
}

func clearTail(targetID string) {
	tailLock.Lock()
	defer tailLock.Unlock()
	delete(tails, targetID)
}

// Run starts the main run loop
func Run(ctx context.Context, config *Config) error {
	clientConfig := kubernetes.NewClientConfig(config.KubeConfig, config.ContextName)
	cc, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	clientset, err := corev1client.NewForConfig(cc)
	if err != nil {
		return err
	}

	var namespaces []string
	// A specific namespace is ignored if all-namespaces is provided
	if config.AllNamespaces {
		namespaces = []string{""}
	} else {
		namespaces = config.Namespaces
		if len(namespaces) == 0 {
			n, _, err := clientConfig.Namespace()
			if err != nil {
				return errors.Wrap(err, "unable to get default namespace")
			}
			namespaces = []string{n}
		}
	}

	added := make(chan *Target)
	removed := make(chan *Target)
	errCh := make(chan error)

	defer close(added)
	defer close(removed)
	defer close(errCh)

	for _, n := range namespaces {
		a, r, err := Watch(ctx,
			clientset.Pods(n),
			config.PodQuery,
			config.ContainerQuery,
			config.ExcludeContainerQuery,
			config.InitContainers,
			config.ContainerState,
			config.LabelSelector,
			config.FieldSelector)
		if err != nil {
			return errors.Wrap(err, "failed to set up watch")
		}

		go func() {
			for {
				select {
				case v, ok := <-a:
					if !ok {
						errCh <- fmt.Errorf("lost watch connection")
						return
					}
					added <- v
				case v, ok := <-r:
					if !ok {
						errCh <- fmt.Errorf("lost watch connection")
						return
					}
					removed <- v
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		for p := range added {
			targetID := p.GetID()

			if tail, ok := getTail(targetID); ok {
				if tail.isActive() {
					continue
				} else {
					tail.Close()
					clearTail(targetID)
				}
			}

			tail := NewTail(clientset, p.Node, p.Namespace, p.Pod, p.Container, config.Template, os.Stdout, os.Stderr, &TailOptions{
				Timestamps:   config.Timestamps,
				SinceSeconds: int64(config.Since.Seconds()),
				Exclude:      config.Exclude,
				Include:      config.Include,
				Namespace:    config.AllNamespaces || len(namespaces) > 1,
				TailLines:    config.TailLines,
			})
			setTail(targetID, tail)

			go func(tail *Tail) {
				if err := tail.Start(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
				}
			}(tail)
		}
	}()

	go func() {
		for p := range removed {
			targetID := p.GetID()
			if tail, ok := getTail(targetID); ok {
				tail.Close()
				clearTail(targetID)
			}
		}
	}()

	select {
	case e := <-errCh:
		return e
	case <-ctx.Done():
		return nil
	}
}
