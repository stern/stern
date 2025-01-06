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
	terminated := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ContainerID: "dummy"}}
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
				InitContainerStatuses: []corev1.ContainerStatus{
					{Name: "init-container1-running", State: running, ContainerID: "dummy"},
					{Name: "init-container2-terminated", State: terminated},
					{Name: "init-container3-waiting", State: waiting, LastTerminationState: terminated},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "container1-running", State: running, ContainerID: "dummy"},
					{Name: "container2-terminated", State: terminated},
					{Name: "container3-waiting", State: waiting, LastTerminationState: terminated},
				},
				EphemeralContainerStatuses: []corev1.ContainerStatus{
					{Name: "ephemeral-container1-running", State: running, ContainerID: "dummy"},
					{Name: "ephemeral-container2-terminated", State: terminated},
					{Name: "ephemeral-container3-waiting", State: waiting, LastTerminationState: terminated},
				},
			},
		}
	}

	pods := []*corev1.Pod{
		createPod("node1", "pod1"),
		createPod("node2", "pod2"),
	}

	genTarget := func(node, pod, container string) Target {
		return Target{
			Namespace: "ns1",
			Node:      node,
			Pod:       pod,
			Container: container,
		}
	}

	tests := []struct {
		name     string
		config   targetFilterConfig
		expected []Target
	}{
		{
			name: "match all",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "init-container1-running"),
				genTarget("node1", "pod1", "init-container2-terminated"),
				genTarget("node1", "pod1", "init-container3-waiting"),
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node1", "pod1", "ephemeral-container1-running"),
				genTarget("node1", "pod1", "ephemeral-container2-terminated"),
				genTarget("node1", "pod1", "ephemeral-container3-waiting"),
				genTarget("node2", "pod2", "init-container1-running"),
				genTarget("node2", "pod2", "init-container2-terminated"),
				genTarget("node2", "pod2", "init-container3-waiting"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
				genTarget("node2", "pod2", "ephemeral-container1-running"),
				genTarget("node2", "pod2", "ephemeral-container2-terminated"),
				genTarget("node2", "pod2", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "filter by podFilter",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`not-matched`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{},
		},
		{
			name: "filter by excludePodFilter",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(``),
				excludePodFilter:       []*regexp.Regexp{regexp.MustCompile(`pod1`)},
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node2", "pod2", "init-container1-running"),
				genTarget("node2", "pod2", "init-container2-terminated"),
				genTarget("node2", "pod2", "init-container3-waiting"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
				genTarget("node2", "pod2", "ephemeral-container1-running"),
				genTarget("node2", "pod2", "ephemeral-container2-terminated"),
				genTarget("node2", "pod2", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "filter by multiple excludePodFilter",
			config: targetFilterConfig{
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
			expected: []Target{
				genTarget("node1", "pod1", "init-container1-running"),
				genTarget("node1", "pod1", "init-container2-terminated"),
				genTarget("node1", "pod1", "init-container3-waiting"),
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node1", "pod1", "ephemeral-container1-running"),
				genTarget("node1", "pod1", "ephemeral-container2-terminated"),
				genTarget("node1", "pod1", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "filter by containerFilter",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*container1.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "init-container1-running"),
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "ephemeral-container1-running"),
				genTarget("node2", "pod2", "init-container1-running"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "ephemeral-container1-running"),
			},
		},
		{
			name: "filter by containerExcludeFilter",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: []*regexp.Regexp{regexp.MustCompile(`.*container1.*`)},
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "init-container2-terminated"),
				genTarget("node1", "pod1", "init-container3-waiting"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node1", "pod1", "ephemeral-container2-terminated"),
				genTarget("node1", "pod1", "ephemeral-container3-waiting"),
				genTarget("node2", "pod2", "init-container2-terminated"),
				genTarget("node2", "pod2", "init-container3-waiting"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
				genTarget("node2", "pod2", "ephemeral-container2-terminated"),
				genTarget("node2", "pod2", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "filter by multiple containerExcludeFilter",
			config: targetFilterConfig{
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
			expected: []Target{
				genTarget("node1", "pod1", "init-container3-waiting"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node1", "pod1", "ephemeral-container2-terminated"),
				genTarget("node1", "pod1", "ephemeral-container3-waiting"),
				genTarget("node2", "pod2", "init-container3-waiting"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
				genTarget("node2", "pod2", "ephemeral-container2-terminated"),
				genTarget("node2", "pod2", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "dot not include initContainers",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         false,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node1", "pod1", "ephemeral-container1-running"),
				genTarget("node1", "pod1", "ephemeral-container2-terminated"),
				genTarget("node1", "pod1", "ephemeral-container3-waiting"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
				genTarget("node2", "pod2", "ephemeral-container1-running"),
				genTarget("node2", "pod2", "ephemeral-container2-terminated"),
				genTarget("node2", "pod2", "ephemeral-container3-waiting"),
			},
		},
		{
			name: "dot not include ephemeralContainers",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    false,
				containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "init-container1-running"),
				genTarget("node1", "pod1", "init-container2-terminated"),
				genTarget("node1", "pod1", "init-container3-waiting"),
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "container2-terminated"),
				genTarget("node1", "pod1", "container3-waiting"),
				genTarget("node2", "pod2", "init-container1-running"),
				genTarget("node2", "pod2", "init-container2-terminated"),
				genTarget("node2", "pod2", "init-container3-waiting"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "container2-terminated"),
				genTarget("node2", "pod2", "container3-waiting"),
			},
		},
		{
			name: "match running states",
			config: targetFilterConfig{
				podFilter:              regexp.MustCompile(`.*`),
				excludePodFilter:       nil,
				containerFilter:        regexp.MustCompile(`.*`),
				containerExcludeFilter: nil,
				initContainers:         true,
				ephemeralContainers:    true,
				containerStates:        []ContainerState{RUNNING},
			},
			expected: []Target{
				genTarget("node1", "pod1", "init-container1-running"),
				genTarget("node1", "pod1", "container1-running"),
				genTarget("node1", "pod1", "ephemeral-container1-running"),
				genTarget("node2", "pod2", "init-container1-running"),
				genTarget("node2", "pod2", "container1-running"),
				genTarget("node2", "pod2", "ephemeral-container1-running"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := []Target{}
			for _, pod := range pods {
				filter := newTargetFilter(tt.config)
				filter.visit(pod, func(target *Target, condition bool) {
					actual = append(actual, *target)
				})
			}

			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v, but actual %v", tt.expected, actual)
			}
		})
	}
}

func TestTargetFilterShouldAdd(t *testing.T) {
	filter := newTargetFilter(targetFilterConfig{
		// matches all
		podFilter:              regexp.MustCompile(`.*`),
		excludePodFilter:       nil,
		containerFilter:        regexp.MustCompile(`.*`),
		containerExcludeFilter: nil,
		initContainers:         true,
		ephemeralContainers:    true,
		containerStates:        []ContainerState{RUNNING, TERMINATED, WAITING},
	})
	createPod := func(cs corev1.ContainerStatus) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "pod1",
				UID:       "uid1",
			},
			Spec: corev1.PodSpec{
				NodeName: "node1",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{cs},
			},
		}
	}
	genTarget := func(container string) Target {
		return Target{
			Namespace: "ns1",
			Node:      "node1",
			Pod:       "pod1",
			Container: container,
		}
	}
	tests := []struct {
		name     string
		forget   bool
		cs       corev1.ContainerStatus
		expected []Target
	}{
		{
			name:     "empty state should be ignored",
			cs:       corev1.ContainerStatus{Name: "c1"},
			expected: []Target{},
		},
		{
			name: "running container observed the first time",
			cs: corev1.ContainerStatus{
				Name:        "c1",
				ContainerID: "cid1",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: []Target{genTarget("c1")},
		},
		{
			name: "same container ID should be ignored",
			cs: corev1.ContainerStatus{
				Name:        "c1",
				ContainerID: "cid1",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: []Target{},
		},
		{
			name: "different container ID can be added",
			cs: corev1.ContainerStatus{
				Name:        "c1",
				ContainerID: "cid2", // changed
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: []Target{genTarget("c1")},
		},
		{
			name:   "forget() allows the same ID ",
			forget: true,
			cs: corev1.ContainerStatus{
				Name:        "c1",
				ContainerID: "cid2",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: []Target{genTarget("c1")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.forget {
				filter.forget("uid1")
			}
			actual := []Target{}
			filter.visit(createPod(tt.cs), func(target *Target, condition bool) {
				actual = append(actual, *target)
			})
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v, but actual %v", tt.expected, actual)
			}
		})
	}
}

func TestChooseContainerID(t *testing.T) {
	lastState := corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{
			ContainerID: "last",
		},
	}
	tests := []struct {
		name     string
		cs       corev1.ContainerStatus
		expected string
	}{
		{
			name: "running",
			cs: corev1.ContainerStatus{
				ContainerID:          "current",
				LastTerminationState: lastState,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: "current",
		},
		{
			name: "running (empty)",
			cs: corev1.ContainerStatus{
				LastTerminationState: lastState,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
			expected: "",
		},
		{
			name: "terminated (current terminated container)",
			cs: corev1.ContainerStatus{
				ContainerID:          "current",
				LastTerminationState: lastState,
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ContainerID: "terminated",
					},
				},
			},
			expected: "terminated",
		},
		{
			name: "terminated (last terminated container)",
			cs: corev1.ContainerStatus{
				ContainerID:          "current",
				LastTerminationState: lastState,
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{},
				},
			},
			expected: "last",
		},
		{
			name: "terminated (empty)",
			cs: corev1.ContainerStatus{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{},
				},
			},
			expected: "",
		},
		{
			name: "waiting",
			cs: corev1.ContainerStatus{
				ContainerID:          "current",
				LastTerminationState: lastState,
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			},
			expected: "last",
		},
		{
			name: "waiting (empty)",
			cs: corev1.ContainerStatus{
				ContainerID: "current", // should be ignored
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			},
			expected: "",
		},
		{
			name: "no current state with last state",
			cs: corev1.ContainerStatus{
				ContainerID:          "current",
				LastTerminationState: lastState,
			},
			expected: "last",
		},
		{
			name:     "empty state",
			cs:       corev1.ContainerStatus{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := chooseContainerID(tt.cs)
			if tt.expected != actual {
				t.Errorf("expected %v, but actual %v", tt.expected, actual)
			}
		})
	}
}
