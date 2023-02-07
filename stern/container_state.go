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
	"errors"

	v1 "k8s.io/api/core/v1"
)

type ContainerState string

const (
	RUNNING    = "running"
	WAITING    = "waiting"
	TERMINATED = "terminated"
	ALL_STATES = "all"
)

// NewContainerState returns corresponding ContainerState
func NewContainerState(stateConfig string) (ContainerState, error) {
	if stateConfig == RUNNING {
		return RUNNING, nil
	} else if stateConfig == WAITING {
		return WAITING, nil
	} else if stateConfig == TERMINATED {
		return TERMINATED, nil
	} else if stateConfig == ALL_STATES {
		return ALL_STATES, nil
	}

	return "", errors.New("containerState should be one of 'running', 'waiting', 'terminated', or 'all'")
}

// Match returns ContainerState is matched
func (stateConfig ContainerState) Match(containerState v1.ContainerState) bool {
	if stateConfig == ALL_STATES {
		return true
	}
	return (stateConfig == RUNNING && containerState.Running != nil) ||
		(stateConfig == WAITING && containerState.Waiting != nil) ||
		(stateConfig == TERMINATED && containerState.Terminated != nil)
}
