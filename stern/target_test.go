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

	createPod := func(node, pod string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      pod,
			},
			Spec: corev1.PodSpec{
				NodeName: node,
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
	}

	pods := []*corev1.Pod{
		createPod("node1", "pod1"),
		createPod("node2", "pod2"),
	}

	type targetMatch struct {
		target                Target
		containerStateMatched bool
	}
	genTargetMatch := func(node, pod, container string, matched bool) targetMatch {
		return targetMatch{
			target: Target{
				Namespace: "ns1",
				Node:      node,
				Pod:       pod,
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
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "init-container1-running", true),
				genTargetMatch("node1", "pod1", "init-container2-terminated", true),
				genTargetMatch("node1", "pod1", "init-container3-waiting", true),
				genTargetMatch("node1", "pod1", "ephemeral-container1-running", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", true),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", true),
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "init-container1-running", true),
				genTargetMatch("node2", "pod2", "init-container2-terminated", true),
				genTargetMatch("node2", "pod2", "init-container3-waiting", true),
				genTargetMatch("node2", "pod2", "ephemeral-container1-running", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", true),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", true),
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
				excludePodFilter:       []*regexp.Regexp{regexp.MustCompile(`pod1`)},
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "init-container1-running", true),
				genTargetMatch("node2", "pod2", "init-container2-terminated", true),
				genTargetMatch("node2", "pod2", "init-container3-waiting", true),
				genTargetMatch("node2", "pod2", "ephemeral-container1-running", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", true),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", true),
			},
		},
		{
			name: "filter by multiple excludePodFilter",
			filter: targetFilter{
				podFilter: regexp.MustCompile(``),
				excludePodFilter: []*regexp.Regexp{
					regexp.MustCompile(`not-matched`),
					regexp.MustCompile(`pod2`),
				},
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "init-container1-running", true),
				genTargetMatch("node1", "pod1", "init-container2-terminated", true),
				genTargetMatch("node1", "pod1", "init-container3-waiting", true),
				genTargetMatch("node1", "pod1", "ephemeral-container1-running", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", true),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", true),
			},
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
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "init-container1-running", true),
				genTargetMatch("node1", "pod1", "ephemeral-container1-running", true),
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "init-container1-running", true),
				genTargetMatch("node2", "pod2", "ephemeral-container1-running", true),
			},
		},
		{
			name: "filter by containerExcludeFilter",
			filter: targetFilter{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: []*regexp.Regexp{regexp.MustCompile(`.*container1.*`)},
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "init-container2-terminated", true),
				genTargetMatch("node1", "pod1", "init-container3-waiting", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", true),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "init-container2-terminated", true),
				genTargetMatch("node2", "pod2", "init-container3-waiting", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", true),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", true),
			},
		},
		{
			name: "filter by multiple containerExcludeFilter",
			filter: targetFilter{
				podFilter:        regexp.MustCompile(`.*`),
				excludePodFilter: nil,
				containerFilter:  regexp.MustCompile(`.*`),
				containerExcludeFilter: []*regexp.Regexp{
					regexp.MustCompile(`.*container1.*`),
					regexp.MustCompile(`init-container2.*`),
				},
				initContainers:      true,
				ephemeralContainers: true,
				containerStates:     []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []targetMatch{
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "init-container3-waiting", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", true),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "init-container3-waiting", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", true),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", true),
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
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "ephemeral-container1-running", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", true),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", true),
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "ephemeral-container1-running", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", true),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", true),
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
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "container2-terminated", true),
				genTargetMatch("node1", "pod1", "container3-waiting", true),
				genTargetMatch("node1", "pod1", "init-container1-running", true),
				genTargetMatch("node1", "pod1", "init-container2-terminated", true),
				genTargetMatch("node1", "pod1", "init-container3-waiting", true),
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "container2-terminated", true),
				genTargetMatch("node2", "pod2", "container3-waiting", true),
				genTargetMatch("node2", "pod2", "init-container1-running", true),
				genTargetMatch("node2", "pod2", "init-container2-terminated", true),
				genTargetMatch("node2", "pod2", "init-container3-waiting", true),
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
				genTargetMatch("node1", "pod1", "container1-running", true),
				genTargetMatch("node1", "pod1", "container2-terminated", false),
				genTargetMatch("node1", "pod1", "container3-waiting", false),
				genTargetMatch("node1", "pod1", "init-container1-running", true),
				genTargetMatch("node1", "pod1", "init-container2-terminated", false),
				genTargetMatch("node1", "pod1", "init-container3-waiting", false),
				genTargetMatch("node1", "pod1", "ephemeral-container1-running", true),
				genTargetMatch("node1", "pod1", "ephemeral-container2-terminated", false),
				genTargetMatch("node1", "pod1", "ephemeral-container3-waiting", false),
				genTargetMatch("node2", "pod2", "container1-running", true),
				genTargetMatch("node2", "pod2", "container2-terminated", false),
				genTargetMatch("node2", "pod2", "container3-waiting", false),
				genTargetMatch("node2", "pod2", "init-container1-running", true),
				genTargetMatch("node2", "pod2", "init-container2-terminated", false),
				genTargetMatch("node2", "pod2", "init-container3-waiting", false),
				genTargetMatch("node2", "pod2", "ephemeral-container1-running", true),
				genTargetMatch("node2", "pod2", "ephemeral-container2-terminated", false),
				genTargetMatch("node2", "pod2", "ephemeral-container3-waiting", false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := []targetMatch{}
			for _, pod := range pods {
				tt.filter.visit(pod, func(target *Target, containerStateMatched bool) {
					actual = append(actual, targetMatch{target: *target, containerStateMatched: containerStateMatched})
				})
			}

			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v, but actual %v", tt.expected, actual)
			}
		})
	}
}
