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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Target is a target to watch
type Target struct {
	Pod       *corev1.Pod
	Container string
}

// GetID returns the ID of the object
func (t *Target) GetID() string {
	return fmt.Sprintf("%s-%s-%s", t.Pod.Namespace, t.Pod.Name, t.Container)
}

// targetState holds a last shown container ID
type targetState struct {
	podUID      string
	containerID string
}

// targetFilter is a filter of Target
type targetFilter struct {
	c            targetFilterConfig
	targetStates map[string]*targetState
	mu           sync.RWMutex
}

type targetFilterConfig struct {
	podFilter              *regexp.Regexp
	excludePodFilter       []*regexp.Regexp
	containerFilter        *regexp.Regexp
	containerExcludeFilter []*regexp.Regexp
	condition              Condition
	initContainers         bool
	ephemeralContainers    bool
	containerStates        []ContainerState
}

func newTargetFilter(c targetFilterConfig) *targetFilter {
	return &targetFilter{
		c:            c,
		targetStates: make(map[string]*targetState),
	}
}

// visit passes filtered Targets to the visitor function
func (f *targetFilter) visit(pod *corev1.Pod, visitor func(t *Target, conditionFound bool)) {
	// filter by pod
	if !f.c.podFilter.MatchString(pod.Name) {
		return
	}

	for _, re := range f.c.excludePodFilter {
		if re.MatchString(pod.Name) {
			return
		}
	}

	// filter by condition
	conditionFound := true
	if f.c.condition != (Condition{}) {
		conditionFound = f.c.condition.Match(pod)
	}

	// filter by container statuses
	var statuses []corev1.ContainerStatus
	if f.c.initContainers {
		// show initContainers first when --no-follow and --max-log-requests 1
		statuses = append(statuses, pod.Status.InitContainerStatuses...)
	}

	statuses = append(statuses, pod.Status.ContainerStatuses...)

	if f.c.ephemeralContainers {
		statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)
	}

OUTER:
	for _, c := range statuses {
		if !f.c.containerFilter.MatchString(c.Name) {
			continue
		}

		for _, re := range f.c.containerExcludeFilter {
			if re.MatchString(c.Name) {
				continue OUTER
			}
		}

		t := &Target{
			Pod:       pod,
			Container: c.Name,
		}

		if !conditionFound {
			visitor(t, false)
			f.forget(string(pod.UID))
			continue
		}

		if f.shouldAdd(t, string(pod.UID), c) {
			visitor(t, true)
		}
	}
}

func (f *targetFilter) matchContainerState(state corev1.ContainerState) bool {
	for _, containerState := range f.c.containerStates {
		if containerState.Match(state) {
			return true
		}
	}
	return false
}

func (f *targetFilter) shouldAdd(t *Target, podUID string, cs corev1.ContainerStatus) bool {
	state := stateToString(cs.State)
	containerID := chooseContainerID(cs)

	f.mu.Lock()
	last := f.targetStates[t.GetID()]
	f.targetStates[t.GetID()] = &targetState{podUID: podUID, containerID: containerID}
	f.mu.Unlock()

	if containerID == "" {
		// does not have a container to retrieve logs
		klog.V(7).InfoS("Container ID is empty", "state", state, "target", t.GetID())
		return false
	}

	if last == nil {
		// We filter out only containers that have existed before stern starts by container states.
		// The container state transition skips the "running" when a pod immediately completes,
		// so filtering by container states does not work as expected for newly created containers.
		klog.V(7).InfoS("Container ID has existed before observation",
			"state", state, "target", t.GetID(), "container", containerID)
		return f.matchContainerState(cs.State)
	}

	if last.containerID == containerID {
		klog.V(7).InfoS("Container ID is the same",
			"state", state, "target", t.GetID(), "container", containerID)
		return false
	}
	// add a container when the container ID is changed from the last time
	klog.V(7).InfoS("Container ID was changed",
		"state", state, "target", t.GetID(), "container", containerID, "last", last.containerID)
	return true
}

func (f *targetFilter) forget(podUID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// delete target states belonging to the pod
	for targetID, state := range f.targetStates {
		if state.podUID == podUID {
			klog.V(7).InfoS("Forget targetState", "target", targetID)
			delete(f.targetStates, targetID)
		}
	}
}

func (f *targetFilter) isActive(t *Target) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	last := f.targetStates[t.GetID()]
	return last != nil && last.containerID != ""
}

func chooseContainerID(cs corev1.ContainerStatus) string {
	// This logic is based on kubelet's validateContainerLogStatus
	// https://github.com/kubernetes/kubernetes/blob/v1.26.1/pkg/kubelet/kubelet_pods.go#L1246
	switch {
	case cs.State.Running != nil:
		return cs.ContainerID
	case cs.State.Terminated != nil:
		if cs.State.Terminated.ContainerID != "" {
			return cs.State.Terminated.ContainerID
		}
	}
	lastTerminated := cs.LastTerminationState.Terminated
	if lastTerminated != nil && lastTerminated.ContainerID != "" {
		return lastTerminated.ContainerID
	}
	return ""
}

func stateToString(state corev1.ContainerState) string {
	switch {
	case state.Running != nil:
		return "running"
	case state.Terminated != nil:
		return "terminated"
	case state.Waiting != nil:
		return "waiting"
	}
	return "unknown"
}
