//   Copyright 2017 Wercker Holding BV
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
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

// Target is a target to watch
type Target struct {
	Node      string
	Namespace string
	Pod       string
	Container string
}

// GetID returns the ID of the object
func (t *Target) GetID() string {
	return fmt.Sprintf("%s-%s-%s", t.Namespace, t.Pod, t.Container)
}

// targetFilter is a filter of Target
type targetFilter struct {
	podFilter              *regexp.Regexp
	excludePodFilter       *regexp.Regexp
	containerFilter        *regexp.Regexp
	containerExcludeFilter *regexp.Regexp
	initContainers         bool
	ephemeralContainers    bool
	containerStates        []ContainerState
}

// visit passes filtered Targets to the visitor function
func (f *targetFilter) visit(pod *corev1.Pod, visitor func(t *Target, containerStateMatched bool)) {
	// filter by pod
	if !f.podFilter.MatchString(pod.Name) {
		return
	}
	if f.excludePodFilter != nil && f.excludePodFilter.MatchString(pod.Name) {
		return
	}

	// filter by container statuses
	var statuses []corev1.ContainerStatus
	statuses = append(statuses, pod.Status.ContainerStatuses...)
	if f.initContainers {
		statuses = append(statuses, pod.Status.InitContainerStatuses...)
	}
	if f.ephemeralContainers {
		statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)
	}
	for _, c := range statuses {
		if !f.containerFilter.MatchString(c.Name) {
			continue
		}
		if f.containerExcludeFilter != nil && f.containerExcludeFilter.MatchString(c.Name) {
			continue
		}
		t := &Target{
			Node:      pod.Spec.NodeName,
			Namespace: pod.Namespace,
			Pod:       pod.Name,
			Container: c.Name,
		}
		visitor(t, f.matchContainerState(c.State))
	}
}

func (f *targetFilter) matchContainerState(state corev1.ContainerState) bool {
	for _, containerState := range f.containerStates {
		if containerState.Match(state) {
			return true
		}
	}
	return false
}
