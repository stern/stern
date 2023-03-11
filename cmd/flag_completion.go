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

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stern/stern/kubernetes"
	"github.com/stern/stern/stern"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

var flagChoices = map[string][]string{
	"color":           []string{"always", "never", "auto"},
	"completion":      []string{"bash", "zsh", "fish"},
	"container-state": []string{stern.RUNNING, stern.WAITING, stern.TERMINATED, stern.ALL_STATES},
	"output":          []string{"default", "raw", "json", "extjson", "ppextjson"},
	"timestamps":      []string{"default", "short"},
}

func runCompletion(shell string, cmd *cobra.Command, out io.Writer) error {
	var err error

	switch shell {
	case "bash":
		err = cmd.GenBashCompletion(out)
	case "zsh":
		err = runCompletionZsh(cmd, out)
	case "fish":
		err = cmd.GenFishCompletion(out, true)
	default:
		err = fmt.Errorf("Unsupported shell type: %q", shell)
	}

	return err
}

// runCompletionZsh is based on `kubectl completion zsh`. This function should
// be replaced by cobra implementation when cobra itself supports zsh completion.
// https://github.com/kubernetes/kubernetes/blob/v1.6.1/pkg/kubectl/cmd/completion.go#L136
func runCompletionZsh(cmd *cobra.Command, out io.Writer) error {
	b := new(bytes.Buffer)
	if err := cmd.GenZshCompletion(b); err != nil {
		return err
	}

	// Cobra doesn't source zsh completion file, explicitly doing it here
	fmt.Fprintf(b, "compdef _stern stern")

	fmt.Fprint(out, b.String())

	return nil
}

func registerCompletionFuncForFlags(cmd *cobra.Command, o *options) error {
	if err := cmd.RegisterFlagCompletionFunc("namespace", namespaceCompletionFunc(o)); err != nil {
		return err
	}

	if err := cmd.RegisterFlagCompletionFunc("node", nodeCompletionFunc(o)); err != nil {
		return err
	}

	if err := cmd.RegisterFlagCompletionFunc("context", contextCompletionFunc(o)); err != nil {
		return err
	}

	// flags with pre-defined choices
	for flag, choices := range flagChoices {
		if err := cmd.RegisterFlagCompletionFunc(flag,
			cobra.FixedCompletions(choices, cobra.ShellCompDirectiveNoFileComp)); err != nil {
			return err
		}
	}

	return nil
}

// namespaceCompletionFunc is a completion function that completes namespaces
// that match the toComplete prefix.
func namespaceCompletionFunc(o *options) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clientConfig := kubernetes.NewClientConfig(o.kubeConfig, o.context)
		clientset, err := kubernetes.NewClientSet(clientConfig)
		if err != nil {
			return compError(err)
		}

		namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return compError(err)
		}

		var comps []string
		for _, ns := range namespaceList.Items {
			if strings.HasPrefix(ns.GetName(), toComplete) {
				comps = append(comps, ns.GetName())
			}
		}

		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// nodeCompletionFunc is a completion function that completes node names
// that match the toComplete prefix.
func nodeCompletionFunc(o *options) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clientConfig := kubernetes.NewClientConfig(o.kubeConfig, o.context)
		clientset, err := kubernetes.NewClientSet(clientConfig)
		if err != nil {
			return compError(err)
		}

		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return compError(err)
		}

		var comps []string
		for _, node := range nodeList.Items {
			if strings.HasPrefix(node.GetName(), toComplete) {
				comps = append(comps, node.GetName())
			}
		}

		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// contextCompletionFunc is a completion function that completes contexts
// that match the toComplete prefix.
func contextCompletionFunc(o *options) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clientConfig := kubernetes.NewClientConfig(o.kubeConfig, o.context)
		config, err := clientConfig.RawConfig()
		if err != nil {
			return compError(err)
		}

		var comps []string
		for name := range config.Contexts {
			if strings.HasPrefix(name, toComplete) {
				comps = append(comps, name)
			}
		}

		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// queryCompletionFunc is a completion function that completes a resource
// that match the toComplete prefix.
func queryCompletionFunc(o *options) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		parts := strings.Split(toComplete, "/")
		if len(parts) != 2 {
			// list available resources in the form "<resource>/"
			for _, matcher := range stern.ResourceMatchers {
				if strings.HasPrefix(matcher.Name(), toComplete) {
					comps = append(comps, matcher.Name()+"/")
				}
			}
			return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
		}

		// list available names in the resources in the form "<resource>/<name>"
		uniqueNamespaces := makeUnique(o.namespaces)
		if o.allNamespaces || len(uniqueNamespaces) > 1 {
			// do not support multiple namespaces for simplicity
			return compError(errors.New("multiple namespaces are not supported"))
		}

		clientConfig := kubernetes.NewClientConfig(o.kubeConfig, o.context)
		clientset, err := kubernetes.NewClientSet(clientConfig)
		if err != nil {
			return compError(err)
		}
		var namespace string
		if len(uniqueNamespaces) == 1 {
			namespace = uniqueNamespaces[0]
		} else {
			n, _, err := clientConfig.Namespace()
			if err != nil {
				return compError(err)
			}
			namespace = n
		}

		kind, name := parts[0], parts[1]
		names, err := retrieveNamesFromResource(context.TODO(), clientset, namespace, kind)
		if err != nil {
			return compError(err)
		}
		for _, n := range names {
			if strings.HasPrefix(n, name) {
				comps = append(comps, kind+"/"+n)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

func compError(err error) ([]string, cobra.ShellCompDirective) {
	cobra.CompError(err.Error())
	return nil, cobra.ShellCompDirectiveError
}

func retrieveNamesFromResource(ctx context.Context, client clientset.Interface, namespace, kind string) ([]string, error) {
	opt := metav1.ListOptions{}
	var names []string
	switch {
	// core
	case stern.PodMatcher.Matches(kind):
		l, err := client.CoreV1().Pods(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	case stern.ReplicationControllerMatcher.Matches(kind):
		l, err := client.CoreV1().ReplicationControllers(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	case stern.ServiceMatcher.Matches(kind):
		l, err := client.CoreV1().Services(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	// apps
	case stern.DeploymentMatcher.Matches(kind):
		l, err := client.AppsV1().Deployments(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	case stern.DaemonSetMatcher.Matches(kind):
		l, err := client.AppsV1().DaemonSets(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	case stern.ReplicaSetMatcher.Matches(kind):
		l, err := client.AppsV1().ReplicaSets(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	case stern.StatefulSetMatcher.Matches(kind):
		l, err := client.AppsV1().StatefulSets(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	// batch
	case stern.JobMatcher.Matches(kind):
		l, err := client.BatchV1().Jobs(namespace).List(ctx, opt)
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			names = append(names, item.GetName())
		}
	default:
		return nil, fmt.Errorf("resource type %s is not supported", kind)
	}
	return names, nil
}
