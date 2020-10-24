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

package kubernetes

import (
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// NewClientConfig returns a new Kubernetes client config set for a context
func NewClientConfig(configPath string, contextName string) clientcmd.ClientConfig {
	configPathList := filepath.SplitList(configPath)
	configLoadingRules := &clientcmd.ClientConfigLoadingRules{}
	if len(configPathList) <= 1 {
		configLoadingRules.ExplicitPath = configPath
	} else {
		configLoadingRules.Precedence = configPathList
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		},
	)
}

// NewClientSet returns a new Kubernetes client for a client config
func NewClientSet(clientConfig clientcmd.ClientConfig) (*kubernetes.Clientset, error) {
	c, err := clientConfig.ClientConfig()

	if err != nil {
		return nil, errors.Wrap(err, "failed to get client config")
	}

	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	return clientset, nil
}
