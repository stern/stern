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

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	"k8s.io/client-go/1.5/pkg/labels"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wercker/stern/stern"

	"github.com/fatih/color"
)

const version = "1.4.0"

type Options struct {
	container     string
	timestamps    bool
	since         time.Duration
	context       string
	namespace     string
	kubeConfig    string
	exclude       []string
	allNamespaces bool
	selector      string
	tail          int64
	color         string
	version       bool
	completion    string
}

var opts = &Options{
	container: ".*",
	tail:      -1,
	color:     "auto",
}

func Run() {
	cmd := &cobra.Command{}
	cmd.Use = "stern pod-query"
	cmd.Short = "Tail multiple pods and containers from Kubernetes"

	cmd.Flags().StringVarP(&opts.container, "container", "c", opts.container, "Container name when multiple containers in pod")
	cmd.Flags().BoolVarP(&opts.timestamps, "timestamps", "t", opts.timestamps, "Print timestamps")
	cmd.Flags().DurationVarP(&opts.since, "since", "s", opts.since, "Return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs.")
	cmd.Flags().StringVar(&opts.context, "context", opts.context, "Kubernetes context to use. Default to current context configured in kubeconfig.")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", opts.namespace, "Kubernetes namespace to use. Default to namespace configured in Kubernetes context")
	cmd.Flags().StringVar(&opts.kubeConfig, "kube-config", "", "Path to kubeconfig file to use")
	cmd.Flags().StringSliceVarP(&opts.exclude, "exclude", "e", opts.exclude, "Regex of log lines to exclude")
	cmd.Flags().BoolVar(&opts.allNamespaces, "all-namespaces", opts.allNamespaces, "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.")
	cmd.Flags().StringVarP(&opts.selector, "selector", "l", opts.selector, "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.")
	cmd.Flags().Int64Var(&opts.tail, "tail", opts.tail, "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.")
	cmd.Flags().StringVar(&opts.color, "color", opts.color, "Color output. Can be 'always', 'never', or 'auto'")
	cmd.Flags().BoolVarP(&opts.version, "version", "v", opts.version, "Print the version and exit")
	cmd.Flags().StringVar(&opts.completion, "completion", opts.completion, "Outputs stern command-line completion code for the specified shell. Can be 'bash' or 'zsh'")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if opts.version {
			fmt.Printf("stern version %s\n", version)
			return nil
		}

		if opts.completion != "" {
			return runCompletion(opts.completion, cmd)
		}

		narg := len(args)
		if (narg > 1) || (narg == 0 && opts.selector == "") {
			return cmd.Help()
		}
		config, err := parseConfig(args)
		if err != nil {
			log.Println(err)
			os.Exit(2)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = stern.Run(ctx, config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return nil
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func parseConfig(args []string) (*stern.Config, error) {
	kubeConfig, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	var podQuery string
	if len(args) == 0 {
		podQuery = ".*"
	} else {
		podQuery = args[0]
	}
	pod, err := regexp.Compile(podQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	container, err := regexp.Compile(opts.container)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	var exclude []*regexp.Regexp

	for _, ex := range opts.exclude {
		rex, err := regexp.Compile(ex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclusion filter")
		}

		exclude = append(exclude, rex)
	}

	var labelSelector labels.Selector
	selector := opts.selector
	if selector == "" {
		labelSelector = labels.Everything()
	} else {
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as label selector")
		}
	}

	var tailLines *int64
	if opts.tail != -1 {
		tailLines = &opts.tail
	}

	colorFlag := opts.color
	if colorFlag == "always" {
		color.NoColor = false
	} else if colorFlag == "never" {
		color.NoColor = true
	} else if colorFlag != "auto" {
		return nil, errors.New("color should be one of 'always', 'never', or 'auto'")
	}

	return &stern.Config{
		KubeConfig:     kubeConfig,
		PodQuery:       pod,
		ContainerQuery: container,
		Exclude:        exclude,
		Timestamps:     opts.timestamps,
		Since:          opts.since,
		ContextName:    opts.context,
		Namespace:      opts.namespace,
		AllNamespaces:  opts.allNamespaces,
		LabelSelector:  labelSelector,
		TailLines:      tailLines,
	}, nil
}

func getKubeConfig() (string, error) {
	var kubeconfig string

	if kubeconfig = os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig, nil
	}

	if kubeconfig = opts.kubeConfig; kubeconfig != "" {
		return kubeconfig, nil
	}

	// kubernetes requires an absolute path
	home, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get user home directory")
	}

	kubeconfig = path.Join(home, ".kube/config")

	return kubeconfig, nil
}
