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

	"github.com/pkg/errors"
	"github.com/wercker/stern/stern"

	cli "gopkg.in/urfave/cli.v1"
)

func Run() {
	app := cli.NewApp()

	app.Name = "stern"
	app.Usage = "Tail multiple pods and containers from Kubernetes"
	app.UsageText = "stern [options] pod-query"
	app.Version = "1.2.0"
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
	}

	app.Action = func(c *cli.Context) error {
		if len(c.Args()) != 1 {
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

	pod, err := regexp.Compile(c.Args()[0])
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

	return &stern.Config{
		KubeConfig:     kubeConfig,
		PodQuery:       pod,
		ContainerQuery: container,
		Exclude:        exclude,
		Timestamps:     c.Bool("timestamps"),
		Since:          c.Duration("since"),
		ContextName:    c.String("context"),
		Namespace:      c.String("namespace"),
	}, nil
}
