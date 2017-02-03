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

	"github.com/pkg/errors"
	"github.com/wercker/stern/kubernetes"
)

// Run starts the main run loop
func Run(ctx context.Context, config *Config) error {
	clientConfig := kubernetes.NewClientConfig(config.KubeConfig, config.ContextName)
	clientset, err := kubernetes.NewClientSet(clientConfig)
	if err != nil {
		return err
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			return errors.Wrap(err, "unable to get default namespace")
		}
	}
	input := clientset.Core().Pods(namespace)

	added, removed, err := Watch(ctx, input, config.PodQuery, config.ContainerQuery)
	if err != nil {
		return errors.Wrap(err, "failed to set up watch")
	}

	tails := make(map[string]*Tail)

	go func() {
		for p := range added {
			ID := id(p.Pod, p.Container)
			if tails[ID] != nil {
				continue
			}

			tail := NewTail(p.Pod, p.Container, &TailOptions{
				Timestamps:   config.Timestamps,
				SinceSeconds: int64(config.Since.Seconds()),
				Exclude:      config.Exclude,
			})
			tails[ID] = tail

			tail.Start(ctx, input)
		}
	}()

	go func() {
		for p := range removed {
			ID := id(p.Pod, p.Container)
			if tails[ID] == nil {
				continue
			}
			tails[ID].Close()
			delete(tails, ID)
		}
	}()

	<-ctx.Done()

	return nil
}

func id(podID string, containerID string) string {
	return fmt.Sprintf("%s-%s", podID, containerID)
}
