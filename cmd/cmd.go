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
	"io"
	"regexp"
	"sort"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type options struct {
	genericclioptions.IOStreams

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
	podQuery            string
}

func NewOptions(streams genericclioptions.IOStreams) *options {
	return &options{
		IOStreams: streams,

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
}

func (o *options) Complete(args []string) error {
	if len(args) > 0 {
		o.podQuery = args[0]
	}

	return nil
}

func (o *options) Validate() error {
	if !o.prompt && o.podQuery == "" && o.selector == "" && o.fieldSelector == "" {
		return errors.New("One of pod-query, --selector, --field-selector or --prompt is required")
	}

	return nil
}

func (o *options) Run(cmd *cobra.Command) error {
	config, err := o.sternConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if o.prompt {
		if err := promptHandler(ctx, config, o.Out); err != nil {
			return err
		}
	}

	return stern.Run(ctx, config)
}

func (o *options) sternConfig() (*stern.Config, error) {
	pod, err := regexp.Compile(o.podQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression from query")
	}

	var excludePod *regexp.Regexp
	if o.excludePod != "" {
		excludePod, err = regexp.Compile(o.excludePod)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular exression for excluded pod query")
		}
	}

	container, err := regexp.Compile(o.container)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	var excludeContainer *regexp.Regexp
	if o.excludeContainer != "" {
		excludeContainer, err = regexp.Compile(o.excludeContainer)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclude container query")
		}
	}

	var exclude []*regexp.Regexp
	for _, ex := range o.exclude {
		rex, err := regexp.Compile(ex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for exclusion filter")
		}

		exclude = append(exclude, rex)
	}

	var include []*regexp.Regexp
	for _, inc := range o.include {
		rin, err := regexp.Compile(inc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile regular expression for inclusion filter")
		}

		include = append(include, rin)
	}

	containerStates := []stern.ContainerState{}
	if o.containerStates != nil {
		for _, containerStateStr := range makeUnique(o.containerStates) {
			containerState, err := stern.NewContainerState(containerStateStr)
			if err != nil {
				return nil, err
			}
			containerStates = append(containerStates, containerState)
		}
	}

	var labelSelector labels.Selector
	if o.selector == "" {
		labelSelector = labels.Everything()
	} else {
		labelSelector, err = labels.Parse(o.selector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as label selector")
		}
	}

	var fieldSelector fields.Selector
	if o.fieldSelector == "" {
		fieldSelector = fields.Everything()
	} else {
		fieldSelector, err = fields.ParseSelector(o.fieldSelector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as field selector")
		}
	}

	var tailLines *int64
	if o.tail != -1 {
		tailLines = &o.tail
	}

	colorFlag := o.color
	if colorFlag == "always" {
		color.NoColor = false
	} else if colorFlag == "never" {
		color.NoColor = true
	} else if colorFlag != "auto" {
		return nil, errors.New("color should be one of 'always', 'never', or 'auto'")
	}

	t := o.template
	if t == "" {
		switch o.output {
		case "default":
			if color.NoColor {
				t = "{{.PodName}} {{.ContainerName}} {{.Message}}"
				if o.allNamespaces || len(o.namespaces) > 1 {
					t = fmt.Sprintf("{{.Namespace}} %s", t)
				}
			} else {
				t = "{{color .PodColor .PodName}} {{color .ContainerColor .ContainerName}} {{.Message}}"
				if o.allNamespaces || len(o.namespaces) > 1 {
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
	if o.namespaces != nil {
		namespaces = makeUnique(o.namespaces)
	}

	// --timezone
	location, err := time.LoadLocation(o.timezone)
	if err != nil {
		return nil, err
	}

	return &stern.Config{
		KubeConfig:            o.kubeConfig,
		ContextName:           o.context,
		Namespaces:            namespaces,
		PodQuery:              pod,
		ExcludePodQuery:       excludePod,
		Timestamps:            o.timestamps,
		Location:              location,
		ContainerQuery:        container,
		ExcludeContainerQuery: excludeContainer,
		ContainerStates:       containerStates,
		Exclude:               exclude,
		Include:               include,
		InitContainers:        o.initContainers,
		EphemeralContainers:   o.ephemeralContainers,
		Since:                 o.since,
		AllNamespaces:         o.allNamespaces,
		LabelSelector:         labelSelector,
		FieldSelector:         fieldSelector,
		TailLines:             tailLines,
		Template:              template,

		Out:    o.Out,
		ErrOut: o.ErrOut,
	}, nil
}

// AddFlags adds all the flags used by stern.
func (o *options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.allNamespaces, "all-namespaces", "A", o.allNamespaces, "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.")
	fs.StringVar(&o.color, "color", o.color, "Force set color output. 'auto':  colorize if tty attached, 'always': always colorize, 'never': never colorize.")
	fs.StringVar(&o.completion, "completion", o.completion, "Output stern command-line completion code for the specified shell. Can be 'bash' or 'zsh'.")
	fs.StringVarP(&o.container, "container", "c", o.container, "Container name when multiple containers in pod. (regular expression)")
	fs.StringSliceVar(&o.containerStates, "container-state", o.containerStates, "Tail containers with state in running, waiting or terminated. To specify multiple states, repeat this or set comma-separated value.")
	fs.StringVar(&o.context, "context", o.context, "Kubernetes context to use. Default to current context configured in kubeconfig.")
	fs.StringSliceVarP(&o.exclude, "exclude", "e", o.exclude, "Log lines to exclude. (regular expression)")
	fs.StringVarP(&o.excludeContainer, "exclude-container", "E", o.excludeContainer, "Container name to exclude when multiple containers in pod. (regular expression)")
	fs.StringVar(&o.excludePod, "exclude-pod", o.excludePod, "Pod name to exclude. (regular expression)")
	fs.StringSliceVarP(&o.include, "include", "i", o.include, "Log lines to include. (regular expression")
	fs.BoolVar(&o.initContainers, "init-containers", o.initContainers, "Include or exclude init containers.")
	fs.BoolVar(&o.ephemeralContainers, "ephemeral-containers", o.ephemeralContainers, "Include or exclude ephemeral containers.")
	fs.StringVar(&o.kubeConfig, "kubeconfig", o.kubeConfig, "Path to kubeconfig file to use. Default to KUBECONFIG variable then ~/.kube/config path.")
	fs.StringVar(&o.kubeConfig, "kube-config", o.kubeConfig, "Path to kubeconfig file to use.")
	_ = fs.MarkDeprecated("kube-config", "Use --kubeconfig instead.")
	fs.StringSliceVarP(&o.namespaces, "namespace", "n", o.namespaces, "Kubernetes namespace to use. Default to namespace configured in kubernetes context. To specify multiple namespaces, repeat this or set comma-separated value.")
	fs.StringVarP(&o.output, "output", "o", o.output, "Specify predefined template. Currently support: [default, raw, json]")
	fs.BoolVarP(&o.prompt, "prompt", "p", o.prompt, "Toggle interactive prompt for selecting 'app.kubernetes.io/instance' label values.")
	fs.StringVarP(&o.selector, "selector", "l", o.selector, "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.StringVar(&o.fieldSelector, "field-selector", o.fieldSelector, "Selector (field query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.DurationVarP(&o.since, "since", "s", o.since, "Return logs newer than a relative duration like 5s, 2m, or 3h.")
	fs.Int64Var(&o.tail, "tail", o.tail, "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.")
	fs.StringVar(&o.template, "template", o.template, "Template to use for log lines, leave empty to use --output flag.")
	fs.BoolVarP(&o.timestamps, "timestamps", "t", o.timestamps, "Print timestamps.")
	fs.StringVar(&o.timezone, "timezone", o.timezone, "Set timestamps to specific timezone.")
	fs.BoolVarP(&o.version, "version", "v", o.version, "Print the version and exit.")
}

func NewSternCmd(stream genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(stream)

	cmd := &cobra.Command{
		Use:   "stern pod-query",
		Short: "Tail multiple pods and containers from Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Output version information and exit
			if o.version {
				fmt.Fprintln(o.Out, buildVersion(version, commit, date))
				return nil
			}

			// Output shell completion code for the specified shell and exit
			if o.completion != "" {
				return runCompletion(o.completion, cmd, o.Out)
			}

			if err := o.Complete(args); err != nil {
				return err
			}

			if err := o.Validate(); err != nil {
				return err
			}

			cmd.SilenceUsage = true

			return o.Run(cmd)
		},
	}

	o.AddFlags(cmd.Flags())

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

	return cmd
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
func promptHandler(ctx context.Context, config *stern.Config, out io.Writer) error {
	labelsMap, err := stern.List(ctx, config)
	if err != nil {
		return err
	}

	if len(labelsMap) == 0 {
		return errors.New("No matching labels")
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

	fmt.Fprintf(out, "Selector: %v\n", color.BlueString(selector))

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
		Message: "Select \"app.kubernetes.io/instance\" label value:",
		Options: pods,
	}

	var pod string

	if err := survey.AskOne(prompt, &pod, arrow); err != nil {
		return "", err
	}

	return pod, nil
}
