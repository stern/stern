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

	"github.com/spf13/cobra"
	"github.com/stern/stern/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	if err := cmd.RegisterFlagCompletionFunc("context", contextCompletionFunc(o)); err != nil {
		return err
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

func compError(err error) ([]string, cobra.ShellCompDirective) {
	cobra.CompError(err.Error())
	return nil, cobra.ShellCompDirectiveError
}
