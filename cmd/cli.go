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
	"os/user"
	"path"
	"regexp"

	"k8s.io/client-go/1.5/pkg/labels"

	"github.com/pkg/errors"
	"github.com/wercker/stern/stern"

	"github.com/fatih/color"
	cli "gopkg.in/urfave/cli.v1"
)

func Run() {
	app := cli.NewApp()

	app.Name = "stern"
	app.Usage = "Tail multiple pods and containers from Kubernetes"
	app.UsageText = "stern [options] pod-query"
	app.Version = "1.4.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "container, c",
			Usage: "Container name when multiple containers in pod",
			Value: ".*",
		},
		cli.BoolFlag{
			Name:  "timestamps, t",
			Usage: "Print timestamps",
		},
		cli.DurationFlag{
			Name:  "since, s",
			Usage: "Return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs.",
		},
		cli.StringFlag{
			Name:  "context",
			Usage: "Kubernetes context to use. Default to `kubectl config current-context`",
			Value: "",
		},
		cli.StringFlag{
			Name:  "namespace, n",
			Usage: "Kubernetes namespace to use. Default to namespace configured in Kubernetes context",
			Value: "",
		},
		cli.StringFlag{
			Name:   "kube-config",
			Usage:  "Path to kubeconfig file to use",
			Value:  "",
			EnvVar: "KUBECONFIG",
		},
		cli.StringSliceFlag{
			Name:  "exclude, e",
			Usage: "Regex of log lines to exclude",
			Value: &cli.StringSlice{},
		},
		cli.BoolFlag{
			Name:  "all-namespaces",
			Usage: "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.",
		},
		cli.StringFlag{
			Name:  "selector, l",
			Usage: "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.",
			Value: "",
		},
		cli.Int64Flag{
			Name:  "tail",
			Usage: "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.",
			Value: -1,
		},
		cli.StringFlag{
			Name:  "color",
			Usage: "Color output. Can be 'always', 'never', or 'auto'",
			Value: "auto",
		},
	}

	app.Action = func(c *cli.Context) error {
		narg := c.NArg()
		if (narg > 1) || (narg == 0 && c.String("selector") == "") {
			return cli.ShowAppHelp(c)
		}

		config, err := parseConfig(c)
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

	app.Run(os.Args)
}

func parseConfig(c *cli.Context) (*stern.Config, error) {
	kubeConfig := c.String("kube-config")
	if kubeConfig == "" {
		// kubernetes requires an absolute path
		u, err := user.Current()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current user")
		}

		kubeConfig = path.Join(u.HomeDir, ".kube/config")
	}

	var podQuery string
	if c.NArg() == 0 {
		podQuery = ".*"
	} else {
		podQuery = c.Args()[0]
	}
	pod, err := regexp.Compile(podQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	container, err := regexp.Compile(c.String("container"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	var exclude []*regexp.Regexp

	for _, ex := range c.StringSlice("exclude") {
		rex, err := regexp.Compile(ex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclusion filter")
		}

		exclude = append(exclude, rex)
	}

	var labelSelector labels.Selector
	selector := c.String("selector")
	if selector == "" {
		labelSelector = labels.Everything()
	} else {
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as label selector")
		}
	}

	var tailLines *int64
	if tail := c.Int64("tail"); tail != -1 {
		tailLines = &tail
	}

	colorFlag := c.String("color")
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
		Timestamps:     c.Bool("timestamps"),
		Since:          c.Duration("since"),
		ContextName:    c.String("context"),
		Namespace:      c.String("namespace"),
		AllNamespaces:  c.Bool("all-namespaces"),
		LabelSelector:  labelSelector,
		TailLines:      tailLines,
	}, nil
}
