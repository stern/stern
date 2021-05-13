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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"text/template"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

type Options struct {
	excludePod          string
	container           string
	excludeContainer    string
	containerStates     []string
	timestamps          bool
	timezone            string
	since               time.Duration
	context             string
	namespaces          []string
	kubeConfig          string
	exclude             []string
	include             []string
	initContainers      bool
	ephemeralContainers bool
	allNamespaces       bool
	selector            string
	fieldSelector       string
	tail                int64
	color               string
	version             bool
	completion          string
	template            string
	output              string
	prompt              bool
}

var opts = &Options{
	color:               "auto",
	container:           ".*",
	containerStates:     []string{"running"},
	initContainers:      true,
	ephemeralContainers: true,
	output:              "default",
	since:               48 * time.Hour,
	tail:                -1,
	template:            "",
	timestamps:          false,
	timezone:            "Local",
	prompt:              false,
}

func Run() {
	cmd := &cobra.Command{}
	cmd.Use = "stern pod-query"
	cmd.Short = "Tail multiple pods and containers from Kubernetes"

	AddFlags(cmd.Flags())

	// Specify custom bash completion function
	cmd.BashCompletionFunction = bash_completion_func
	for name, completion := range bash_completion_flags {
		if cmd.Flag(name) != nil {
			if cmd.Flag(name).Annotations == nil {
				cmd.Flag(name).Annotations = map[string][]string{}
			}
			cmd.Flag(name).Annotations[cobra.BashCompCustom] = append(
				cmd.Flag(name).Annotations[cobra.BashCompCustom],
				completion,
			)
		}
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if opts.version {
			fmt.Println(buildVersion(version, commit, date))

			return nil
		}

		if opts.completion != "" {
			return runCompletion(opts.completion, cmd)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		narg := len(args)

		if (narg > 1) || (narg == 0 && opts.selector == "" && opts.fieldSelector == "") && !opts.prompt {
			return cmd.Help()
		}

		config, err := parseConfig(args)
		if err != nil {
			log.Println(err)
			os.Exit(2)
		}

		if opts.prompt {
			if err := promptHandler(ctx, config); err != nil {
				return err
			}
		}

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

// AddFlags adds all the flags used by stern.
func AddFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.allNamespaces, "all-namespaces", "A", opts.allNamespaces, "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.")
	fs.StringVar(&opts.color, "color", opts.color, "Force set color output. 'auto':  colorize if tty attached, 'always': always colorize, 'never': never colorize.")
	fs.StringVar(&opts.completion, "completion", opts.completion, "Output stern command-line completion code for the specified shell. Can be 'bash' or 'zsh'.")
	fs.StringVarP(&opts.container, "container", "c", opts.container, "Container name when multiple containers in pod. (regular expression)")
	fs.StringSliceVar(&opts.containerStates, "container-state", opts.containerStates, "Tail containers with state in running, waiting or terminated. To specify multiple states, repeat this or set comma-separated value.")
	fs.StringVar(&opts.context, "context", opts.context, "Kubernetes context to use. Default to current context configured in kubeconfig.")
	fs.StringSliceVarP(&opts.exclude, "exclude", "e", opts.exclude, "Log lines to exclude. (regular expression)")
	fs.StringVarP(&opts.excludeContainer, "exclude-container", "E", opts.excludeContainer, "Container name to exclude when multiple containers in pod. (regular expression)")
	fs.StringVar(&opts.excludePod, "exclude-pod", opts.excludePod, "Pod name to exclude. (regular expression)")
	fs.StringSliceVarP(&opts.include, "include", "i", opts.include, "Log lines to include. (regular expression")
	fs.BoolVar(&opts.initContainers, "init-containers", opts.initContainers, "Include or exclude init containers.")
	fs.BoolVar(&opts.ephemeralContainers, "ephemeral-containers", opts.ephemeralContainers, "Include or exclude ephemeral containers.")
	fs.StringVar(&opts.kubeConfig, "kubeconfig", opts.kubeConfig, "Path to kubeconfig file to use. Default to KUBECONFIG variable then ~/.kube/config path.")
	fs.StringVar(&opts.kubeConfig, "kube-config", opts.kubeConfig, "Path to kubeconfig file to use.")
	_ = fs.MarkDeprecated("kube-config", "Use --kubeconfig instead.")
	fs.StringSliceVarP(&opts.namespaces, "namespace", "n", opts.namespaces, "Kubernetes namespace to use. Default to namespace configured in kubernetes context. To specify multiple namespaces, repeat this or set comma-separated value.")
	fs.StringVarP(&opts.output, "output", "o", opts.output, "Specify predefined template. Currently support: [default, raw, json]")
	fs.BoolVarP(&opts.prompt, "prompt", "p", opts.prompt, "Toggle interactive prompt for selecting 'app.kubernetes.io/instance' label values.")
	fs.StringVarP(&opts.selector, "selector", "l", opts.selector, "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.StringVar(&opts.fieldSelector, "field-selector", opts.fieldSelector, "Selector (field query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.DurationVarP(&opts.since, "since", "s", opts.since, "Return logs newer than a relative duration like 5s, 2m, or 3h.")
	fs.Int64Var(&opts.tail, "tail", opts.tail, "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.")
	fs.StringVar(&opts.template, "template", opts.template, "Template to use for log lines, leave empty to use --output flag.")
	fs.BoolVarP(&opts.timestamps, "timestamps", "t", opts.timestamps, "Print timestamps.")
	fs.StringVar(&opts.timezone, "timezone", opts.timezone, "Set timestamps to specific timezone.")
	fs.BoolVarP(&opts.version, "version", "v", opts.version, "Print the version and exit.")
}

func parseConfig(args []string) (*stern.Config, error) {
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

	var excludePod *regexp.Regexp
	if opts.excludePod != "" {
		excludePod, err = regexp.Compile(opts.excludePod)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular exression for excluded pod query")
		}
	}

	container, err := regexp.Compile(opts.container)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	var excludeContainer *regexp.Regexp
	if opts.excludeContainer != "" {
		excludeContainer, err = regexp.Compile(opts.excludeContainer)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclude container query")
		}
	}

	var exclude []*regexp.Regexp
	for _, ex := range opts.exclude {
		rex, err := regexp.Compile(ex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclusion filter")
		}

		exclude = append(exclude, rex)
	}

	var include []*regexp.Regexp
	for _, inc := range opts.include {
		rin, err := regexp.Compile(inc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for inclusion filter")
		}

		include = append(include, rin)
	}

	containerStates := []stern.ContainerState{}
	if opts.containerStates != nil {
		for _, containerStateStr := range makeUnique(opts.containerStates) {
			containerState, err := stern.NewContainerState(containerStateStr)
			if err != nil {
				return nil, err
			}
			containerStates = append(containerStates, containerState)
		}
	}

	var labelSelector labels.Selector
	if opts.selector == "" {
		labelSelector = labels.Everything()
	} else {
		labelSelector, err = labels.Parse(opts.selector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as label selector")
		}
	}

	var fieldSelector fields.Selector
	if opts.fieldSelector == "" {
		fieldSelector = fields.Everything()
	} else {
		fieldSelector, err = fields.ParseSelector(opts.fieldSelector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as field selector")
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

	t := opts.template
	if t == "" {
		switch opts.output {
		case "default":
			if color.NoColor {
				t = "{{.PodName}} {{.ContainerName}} {{.Message}}"
				if opts.allNamespaces || len(opts.namespaces) > 1 {
					t = fmt.Sprintf("{{.Namespace}} %s", t)
				}
			} else {
				t = "{{color .PodColor .PodName}} {{color .ContainerColor .ContainerName}} {{.Message}}"
				if opts.allNamespaces || len(opts.namespaces) > 1 {
					t = fmt.Sprintf("{{color .PodColor .Namespace}} %s", t)
				}

			}
		case "raw":
			t = "{{.Message}}"
		case "json":
			t = "{{json .}}"
		}
		t += "\n"
	}

	funs := map[string]interface{}{
		"json": func(in interface{}) (string, error) {
			b, err := json.Marshal(in)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"color": func(color color.Color, text string) string {
			return color.SprintFunc()(text)
		},
	}
	template, err := template.New("log").Funcs(funs).Parse(t)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse template")
	}

	namespaces := []string{}
	if opts.namespaces != nil {
		namespaces = makeUnique(opts.namespaces)
	}

	// --timezone
	location, err := time.LoadLocation(opts.timezone)
	if err != nil {
		return nil, err
	}

	return &stern.Config{
		KubeConfig:            opts.kubeConfig,
		ContextName:           opts.context,
		Namespaces:            namespaces,
		PodQuery:              pod,
		ExcludePodQuery:       excludePod,
		Timestamps:            opts.timestamps,
		Location:              location,
		ContainerQuery:        container,
		ExcludeContainerQuery: excludeContainer,
		ContainerStates:       containerStates,
		Exclude:               exclude,
		Include:               include,
		InitContainers:        opts.initContainers,
		EphemeralContainers:   opts.ephemeralContainers,
		Since:                 opts.since,
		AllNamespaces:         opts.allNamespaces,
		LabelSelector:         labelSelector,
		FieldSelector:         fieldSelector,
		TailLines:             tailLines,
		Template:              template,
	}, nil
}

func buildVersion(version, commit, date string) string {
	result := fmt.Sprintf("version: %s", version)

	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}

	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}

	return result
}

// makeUnique makes items in string slice unique
func makeUnique(items []string) []string {
	result := []string{}
	m := make(map[string]struct{})

	for _, item := range items {
		if item == "" {
			continue
		}

		if _, ok := m[item]; !ok {
			m[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

// promptHandler invokes the interactive prompt and updates config.LabelSelector with the selected value.
func promptHandler(ctx context.Context, config *stern.Config) error {
	labelsMap, err := stern.List(ctx, config)
	if err != nil {
		return err
	}

	if len(labelsMap) == 0 {
		log.Fatal("No matching labels.")
	}

	var choices []string

	for key := range labelsMap {
		choices = append(choices, key)
	}

	sort.Strings(choices)

	choice, err := selectPods(choices)
	if err != nil {
		return err
	}

	selector := fmt.Sprintf("%v=%v", labelsMap[choice], choice)

	fmt.Printf("Selector: %v\n", color.BlueString(selector))

	labelSelector, err := labels.Parse(selector)
	if err != nil {
		return err
	}

	config.LabelSelector = labelSelector

	return nil
}

// selectPods surfaces an interactive prompt for selecting an app.kubernetes.io/instance.
func selectPods(pods []string) (string, error) {
	arrow := survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = "❯"
		icons.SelectFocus.Text = "❯"
		icons.Question.Format = "blue"
		icons.SelectFocus.Format = "blue"
	})

	prompt := &survey.Select{
		Message: "Select label value:",
		Options: pods,
	}

	var pod string

	if err := survey.AskOne(prompt, &pod, arrow); err != nil {
		return "", err
	}

	return pod, nil
}
