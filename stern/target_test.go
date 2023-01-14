package stern

import (
	"reflect"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTargetFilter(t *testing.T) {
	running := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	terminated := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
	waiting := corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "pod1",
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container1-running", State: running},
				{Name: "container2-terminated", State: terminated},
				{Name: "container3-waiting", State: waiting},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{Name: "init-container1-running", State: running},
				{Name: "init-container2-terminated", State: terminated},
				{Name: "init-container3-waiting", State: waiting},
			},
			EphemeralContainerStatuses: []corev1.ContainerStatus{
				{Name: "ephemeral-container1-running", State: running},
				{Name: "ephemeral-container2-terminated", State: terminated},
				{Name: "ephemeral-container3-waiting", State: waiting},
			},
		},
	}

	type targetMatch struct {
		target                Target
		containerStateMatched bool
	}
	genTargetMatch := func(container string, matched bool) targetMatch {
		return targetMatch{
			target: Target{
				Node:      "node1",
				Namespace: "ns1",
				Pod:       "pod1",
				Container: container,
			},
			containerStateMatched: matched,
		}
	}

	tests := []struct {
		name     string
		filter   targetFilter
		expected []targetMatch
	}{
		{
			name: "match all",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("container1-running", true),
				genTargetMatch("container2-terminated", true),
				genTargetMatch("container3-waiting", true),
				genTargetMatch("init-container1-running", true),
				genTargetMatch("init-container2-terminated", true),
				genTargetMatch("init-container3-waiting", true),
				genTargetMatch("ephemeral-container1-running", true),
				genTargetMatch("ephemeral-container2-terminated", true),
				genTargetMatch("ephemeral-container3-waiting", true),
			},
		},
		{
			name: "filter by podFilter",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`not-matched`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{},
		},
		{
			name: "filter by excludePodFilter",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(``),
				excludePodFilter:       regexp.MustCompile(`pod1`),
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{},
		},
		{
			name: "filter by containerFilter",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*container1.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("container1-running", true),
				genTargetMatch("init-container1-running", true),
				genTargetMatch("ephemeral-container1-running", true),
			},
		},
		{
			name: "filter by containerExcludeFilter",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: regexp.MustCompile(`.*container1.*`),
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("container2-terminated", true),
				genTargetMatch("container3-waiting", true),
				genTargetMatch("init-container2-terminated", true),
				genTargetMatch("init-container3-waiting", true),
				genTargetMatch("ephemeral-container2-terminated", true),
				genTargetMatch("ephemeral-container3-waiting", true),
			},
		},
		{
			name: "dot not include initContainers",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         false,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("container1-running", true),
				genTargetMatch("container2-terminated", true),
				genTargetMatch("container3-waiting", true),
				genTargetMatch("ephemeral-container1-running", true),
				genTargetMatch("ephemeral-container2-terminated", true),
				genTargetMatch("ephemeral-container3-waiting", true),
			},
		},
		{
			name: "dot not include ephemeralContainers",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    false,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("container1-running", true),
				genTargetMatch("container2-terminated", true),
				genTargetMatch("container3-waiting", true),
				genTargetMatch("init-container1-running", true),
				genTargetMatch("init-container2-terminated", true),
				genTargetMatch("init-container3-waiting", true),
			},
		},
		{
			name: "match running states",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING},
			},
			expected: []targetMatch{
				genTargetMatch("container1-running", true),
				genTargetMatch("container2-terminated", false),
				genTargetMatch("container3-waiting", false),
				genTargetMatch("init-container1-running", true),
				genTargetMatch("init-container2-terminated", false),
				genTargetMatch("init-container3-waiting", false),
				genTargetMatch("ephemeral-container1-running", true),
				genTargetMatch("ephemeral-container2-terminated", false),
				genTargetMatch("ephemeral-container3-waiting", false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := []targetMatch{}
			tt.filter.visit(pod, func(target *Target, containerStateMatched bool) {
				actual = append(actual, targetMatch{target: *target, containerStateMatched: containerStateMatched})
			})
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v, but actual %v", tt.expected, actual)
			}
		})
	}
}
