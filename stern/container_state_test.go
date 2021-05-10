package stern

import (
	"testing"

	"k8s.io/api/core/v1"
)

func TestNewContainerState(t *testing.T) {
	tests := []struct {
		stateConfig string
		expected    ContainerState
		isError     bool
	}{
		{
			"running",
			ContainerState(RUNNING),
			false,
		},
		{
			"waiting",
			ContainerState(WAITING),
			false,
		},
		{
			"terminated",
			ContainerState(TERMINATED),
			false,
		},
		{
			"wrongValue",
			ContainerState(""),
			true,
		},
	}

	for i, tt := range tests {
		containerState, err := NewContainerState(tt.stateConfig)

		if tt.expected != containerState {
			t.Errorf("%d: expected %v, but actual %v", i, tt.expected, containerState)
		}

		if (tt.isError && err == nil) || (!tt.isError && err != nil) {
			t.Errorf("%d: expected error is %v, but actual %v", i, tt.isError, err)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		containerState   ContainerState
		v1ContainerState v1.ContainerState
		expected         bool
	}{
		{
			ContainerState(RUNNING),
			v1.ContainerState{
				Running:    &v1.ContainerStateRunning{},
				Waiting:    nil,
				Terminated: nil,
			},
			true,
		},
		{
			ContainerState(WAITING),
			v1.ContainerState{
				Running:    nil,
				Waiting:    &v1.ContainerStateWaiting{},
				Terminated: nil,
			},
			true,
		},
		{
			ContainerState(TERMINATED),
			v1.ContainerState{
				Running:    nil,
				Waiting:    nil,
				Terminated: &v1.ContainerStateTerminated{},
			},
			true,
		},
		{
			ContainerState(RUNNING),
			v1.ContainerState{
				Running:    nil,
				Waiting:    &v1.ContainerStateWaiting{},
				Terminated: nil,
			},
			false,
		},
	}

	for i, tt := range tests {
		actual := tt.containerState.Match(tt.v1ContainerState)

		if tt.expected != actual {
			t.Errorf("%d: expected %v, but actual %v", i, tt.expected, actual)
		}
	}
}
