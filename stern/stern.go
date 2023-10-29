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
	"regexp"
	"strings"
	"time"

	"sync/atomic"

	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Run starts the main run loop
func Run(ctx context.Context, client kubernetes.Interface, config *Config) error {
	var namespaces []string
	// A specific namespace is ignored if all-namespaces is provided
	if config.AllNamespaces {
		namespaces = []string{""}
	} else {
		namespaces = config.Namespaces
		if len(namespaces) == 0 {
			return errors.New("no namespace specified")
		}
	}

	var resource struct {
		kind string
		name string
	}
	if config.Resource != "" {
		parts := strings.Split(config.Resource, "/")
		if len(parts) != 2 {
			return errors.New("resource must be specified in the form \"<resource>/<name>\"")
		}
		resource.kind, resource.name = parts[0], parts[1]
		if PodMatcher.Matches(resource.kind) {
			// Pods might have no labels or share the same labels,
			// so we use an exact match instead.
			podName, err := regexp.Compile("^" + resource.name + "$")
			if err != nil {
				return errors.Wrap(err, "failed to compile regular expression for pod")
			}
			config.PodQuery = podName
		}
	}

	filter := newTargetFilter(targetFilterConfig{
		podFilter:              config.PodQuery,
		excludePodFilter:       config.ExcludePodQuery,
		containerFilter:        config.ContainerQuery,
		containerExcludeFilter: config.ExcludeContainerQuery,
		initContainers:         config.InitContainers,
		ephemeralContainers:    config.EphemeralContainers,
		containerStates:        config.ContainerStates,
	})
	newTail := func(t *Target) *Tail {
		return NewTail(client.CoreV1(), t.Node, t.Namespace, t.Pod, t.Container, config.Template, config.Out, config.ErrOut, &TailOptions{
			Timestamps:      config.Timestamps,
			TimestampFormat: config.TimestampFormat,
			Location:        config.Location,
			SinceSeconds:    ptr.To[int64](int64(config.Since.Seconds())),
			Exclude:         config.Exclude,
			Include:         config.Include,
			Namespace:       config.AllNamespaces || len(namespaces) > 1,
			TailLines:       config.TailLines,
			Follow:          config.Follow,
			OnlyLogLines:    config.OnlyLogLines,
		})
	}

	if !config.Follow {
		var eg errgroup.Group
		eg.SetLimit(config.MaxLogRequests)
		for _, n := range namespaces {
			selector, err := chooseSelector(ctx, client, n, resource.kind, resource.name, config.LabelSelector)
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

	tailTarget := func(ctx context.Context, target *Target) {
		// We use a rate limiter to prevent a burst of retries.
		// It also enables us to retry immediately, in most cases,
		// when it is disconnected on the way.
		limiter := rate.NewLimiter(rate.Every(time.Second*20), 2)
		var resumeRequest *ResumeRequest
		for {
			if err := limiter.Wait(ctx); err != nil {
				fmt.Fprintf(config.ErrOut, "failed to retry: %v\n", err)
				return
			}
			tail := newTail(target)
			var err error
			if resumeRequest == nil {
				err = tail.Start(ctx)
			} else {
				err = tail.Resume(ctx, resumeRequest)
			}
			tail.Close()
			if err == nil {
				return
			}
			if !filter.isActive(target) {
				fmt.Fprintf(config.ErrOut, "failed to tail: %v\n", err)
				return
			}
			fmt.Fprintf(config.ErrOut, "failed to tail: %v, will retry\n", err)
			if resumeReq := tail.GetResumeRequest(); resumeReq != nil {
				resumeRequest = resumeReq
			}
		}
	}

	eg, nctx := errgroup.WithContext(ctx)
	var numRequests atomic.Int64
	for _, n := range namespaces {
		selector, err := chooseSelector(nctx, client, n, resource.kind, resource.name, config.LabelSelector)
		if err != nil {
			return err
		}
		a, err := WatchTargets(nctx,
			client.CoreV1().Pods(n),
			selector,
			config.FieldSelector,
			filter,
		)
		if err != nil {
			return errors.Wrap(err, "failed to set up watch")
		}

		eg.Go(func() error {
			for {
				select {
				case target, ok := <-a:
					if !ok {
						return fmt.Errorf("lost watch connection")
					}
					numRequests.Add(1)
					if numRequests.Load() > int64(config.MaxLogRequests) {
						return fmt.Errorf(
							"stern reached the maximum number of log requests (%d),"+
								" use --max-log-requests to increase the limit",
							config.MaxLogRequests)
					}
					go func() {
						tailTarget(nctx, target)
						numRequests.Add(-1)
					}()
				case <-nctx.Done():
					return nil
				}
			}
		})
	}
	return eg.Wait()
}

func chooseSelector(ctx context.Context, client kubernetes.Interface, namespace, kind, name string, selector labels.Selector) (labels.Selector, error) {
	if kind == "" {
		return selector, nil
	}
	if PodMatcher.Matches(kind) {
		// We use an exact match for pods instead of a label to select pods without labels.
		return labels.Everything(), nil
	}
	labelMap, err := retrieveLabelsFromResource(ctx, client, namespace, kind, name)
	if err != nil {
		return nil, err
	}
	if len(labelMap) == 0 {
		return nil, fmt.Errorf("resource %s/%s has no labels to select", kind, name)
	}
	return labels.SelectorFromSet(labelMap), nil
}

func retrieveLabelsFromResource(ctx context.Context, client kubernetes.Interface, namespace, kind, name string) (map[string]string, error) {
	opt := metav1.GetOptions{}
	switch {
	// core
	case ReplicationControllerMatcher.Matches(kind):
		o, err := client.CoreV1().ReplicationControllers(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		if o.Spec.Template == nil { // RC's spec.template is a pointer field
			return nil, fmt.Errorf("%s does not have spec.template", name)
		}
		return o.Spec.Template.Labels, nil
	case ServiceMatcher.Matches(kind):
		o, err := client.CoreV1().Services(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Selector, nil
	// apps
	case DaemonSetMatcher.Matches(kind):
		o, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case DeploymentMatcher.Matches(kind):
		o, err := client.AppsV1().Deployments(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case ReplicaSetMatcher.Matches(kind):
		o, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	case StatefulSetMatcher.Matches(kind):
		o, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	// batch
	// We do not support cronjobs because they might not have labels to select.
	case JobMatcher.Matches(kind):
		o, err := client.BatchV1().Jobs(namespace).Get(ctx, name, opt)
		if err != nil {
			return nil, err
		}
		return o.Spec.Template.Labels, nil
	}
	return nil, fmt.Errorf("resource type %s is not supported", kind)
}
