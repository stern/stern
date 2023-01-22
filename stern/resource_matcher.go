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

// ResourceMatcher is a matcher for Kubernetes resources
type ResourceMatcher struct {
	name    string   // the resource name in singular e.g. "deployment"
	aliases []string // the aliases of the resource e.g. "deploy" and "deployments"
}

// Name returns the resource name in singular
func (r *ResourceMatcher) Name() string {
	return r.name
}

// AllNames returns the resource names including the aliases
func (r *ResourceMatcher) AllNames() []string {
	return append(r.aliases, r.name)
}

// Matches returns if name matches one of the resource names
func (r *ResourceMatcher) Matches(name string) bool {
	for _, n := range r.AllNames() {
		if n == name {
			return true
		}
	}
	return false
}

var (
	PodMatcher                   = ResourceMatcher{name: "pod", aliases: []string{"po", "pods"}}
	ReplicationControllerMatcher = ResourceMatcher{name: "replicationcontroller", aliases: []string{"rc", "replicationcontrollers"}}
	ServiceMatcher               = ResourceMatcher{name: "service", aliases: []string{"svc", "services"}}
	DaemonSetMatcher             = ResourceMatcher{name: "daemonset", aliases: []string{"ds", "daemonsets"}}
	DeploymentMatcher            = ResourceMatcher{name: "deployment", aliases: []string{"deploy", "deployments"}}
	ReplicaSetMatcher            = ResourceMatcher{name: "replicaset", aliases: []string{"rs", "replicasets"}}
	StatefulSetMatcher           = ResourceMatcher{name: "statefulset", aliases: []string{"sts", "statefulsets"}}
	JobMatcher                   = ResourceMatcher{name: "job", aliases: []string{"jobs"}} // job does not have a short name
	ResourceMatchers             = []ResourceMatcher{
		PodMatcher,
		ReplicationControllerMatcher,
		ServiceMatcher,
		DeploymentMatcher,
		DaemonSetMatcher,
		ReplicaSetMatcher,
		StatefulSetMatcher,
		JobMatcher,
	}
)
