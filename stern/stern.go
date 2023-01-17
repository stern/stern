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
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/stern/stern/kubernetes"
	"golang.org/x/sync/errgroup"

	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
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

	client, err := clientset.NewForConfig(cc)
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

	filter := &targetFilter{
		podFilter:              config.PodQuery,
		excludePodFilter:       config.ExcludePodQuery,
		containerFilter:        config.ContainerQuery,
		containerExcludeFilter: config.ExcludeContainerQuery,
		initContainers:         config.InitContainers,
		ephemeralContainers:    config.EphemeralContainers,
		containerStates:        config.ContainerStates,
	}
	newTail := func(t *Target) *Tail {
		return NewTail(client.CoreV1(), t.Node, t.Namespace, t.Pod, t.Container, config.Template, config.Out, config.ErrOut, &TailOptions{
			Timestamps:   config.Timestamps,
			Location:     config.Location,
			SinceSeconds: int64(config.Since.Seconds()),
			Exclude:      config.Exclude,
			Include:      config.Include,
			Namespace:    config.AllNamespaces || len(namespaces) > 1,
			TailLines:    config.TailLines,
			Follow:       config.Follow,
		})
	}

	if !config.Follow {
		var eg errgroup.Group
		for _, n := range namespaces {
			selector, err := chooseSelector(ctx, client, n, config.Resource, config.LabelSelector)
			if err != nil {
				return err
			}
			targets, err := ListTargets(ctx,
				client.CoreV1().Pods(n),
				selector,
				config.FieldSelector,
				filter,
			)
			if err != nil {
				return err
			}
			for _, t := range targets {
				t := t
				eg.Go(func() error {
					tail := newTail(t)
					defer tail.Close()
					return tail.Start(ctx)
				})
			}
		}
		return eg.Wait()
	}

	added := make(chan *Target)
	removed := make(chan *Target)
	errCh := make(chan error)

	defer close(added)
	defer close(removed)
	defer close(errCh)

	for _, n := range namespaces {
		selector, err := chooseSelector(ctx, client, n, config.Resource, config.LabelSelector)
		if err != nil {
			return err
		}
		a, r, err := WatchTargets(ctx,
			client.CoreV1().Pods(n),
			selector,
			config.FieldSelector,
			filter,
		)
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

			tail := newTail(p)
			setTail(targetID, tail)

			go func(tail *Tail) {
				if err := tail.Start(ctx); err != nil {
					fmt.Fprintf(config.ErrOut, "unexpected error: %v\n", err)
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

func chooseSelector(ctx context.Context, client clientset.Interface, namespace, resource string, selector labels.Selector) (labels.Selector, error) {
	if resource == "" {
		return selector, nil
	}
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return nil, errors.New("resource must be specified in the form \"<resource>/<name>\"")
	}
	labelMap, err := retrieveLabelsFromResource(ctx, client, namespace, parts[0], parts[1])
	if err != nil {
		return nil, err
	}
	if len(labelMap) == 0 {
		return nil, fmt.Errorf("resource %s has no labels", resource)
	}
	return labels.SelectorFromSet(labelMap), nil
}

func retrieveLabelsFromResource(ctx context.Context, client clientset.Interface, namespace, kind, name string) (map[string]string, error) {
	opt := metav1.GetOptions{}
	switch kind {
	// core
	case "po", "pods", "pod":
		o, err := client.CoreV1().Pods(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Labels, nil
	case "rc", "replicationcontrollers", "replicationcontroller":
		o, err := client.CoreV1().ReplicationControllers(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		if o.Spec.Template == nil { // RC's spec.template is a pointer field
			return nil, fmt.Errorf("%s does not have spec.template", name)
		}
		return o.Spec.Template.Labels, nil
	// apps
	case "deploy", "deployments", "deployment":
		o, err := client.AppsV1().Deployments(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case "ds", "daemonsets", "daemonset":
		o, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case "rs", "replicasets", "replicaset":
		o, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case "sts", "statefulsets", "statefulset":
		o, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	// batch
	case "jobs", "job": // job does not have a short name
		o, err := client.BatchV1().Jobs(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case "cj", "cronjobs", "cronjob":
		o, err := client.BatchV1().CronJobs(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.JobTemplate.Spec.Template.Labels, nil
	}
	return nil, fmt.Errorf("unknown resource type %s", kind)
}
