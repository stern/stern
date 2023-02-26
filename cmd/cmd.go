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
	goflag "flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"k8s.io/klog/v2"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type options struct {
	genericclioptions.IOStreams

	excludePod          []string
	container           string
	excludeContainer    []string
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
	templateFile        string
	output              string
	prompt              bool
	podQuery            string
	noFollow            bool
	resource            string
	verbosity           int
	onlyLogLines        bool
	maxLogRequests      int
}

func NewOptions(streams genericclioptions.IOStreams) *options {
	return &options{
		IOStreams: streams,

		color:               "auto",
		container:           ".*",
		containerStates:     []string{stern.ALL_STATES},
		initContainers:      true,
		ephemeralContainers: true,
		output:              "default",
		since:               48 * time.Hour,
		tail:                -1,
		template:            "",
		templateFile:        "",
		timestamps:          false,
		timezone:            "Local",
		prompt:              false,
		noFollow:            false,
		maxLogRequests:      -1,
	}
}

func (o *options) Complete(args []string) error {
	if len(args) > 0 {
		if s := args[0]; strings.Contains(s, "/") {
			o.resource = s
		} else {
			o.podQuery = s
		}
	}

	return nil
}

func (o *options) Validate() error {
	if !o.prompt && o.podQuery == "" && o.resource == "" && o.selector == "" && o.fieldSelector == "" {
		return errors.New("One of pod-query, --selector, --field-selector or --prompt is required")
	}
	if o.selector != "" && o.resource != "" {
		return errors.New("--selector and the <resource>/<name> query can not be set at the same time")
	}

	return nil
}

func (o *options) Run(cmd *cobra.Command) error {
	if err := o.setVerbosity(); err != nil {
		return err
	}

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

	excludePod, err := compileREs(o.excludePod)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for excluded pod query")
	}

	container, err := regexp.Compile(o.container)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for container query")
	}

	excludeContainer, err := compileREs(o.excludeContainer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for excluded container query")
	}

	exclude, err := compileREs(o.exclude)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for exclusion filter")
	}

	include, err := compileREs(o.include)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for inclusion filter")
	}

	containerStates := []stern.ContainerState{}
	for _, containerStateStr := range makeUnique(o.containerStates) {
		containerState, err := stern.NewContainerState(containerStateStr)
		if err != nil {
			return nil, err
		}
		containerStates = append(containerStates, containerState)
	}

	labelSelector := labels.Everything()
	if o.selector != "" {
		labelSelector, err = labels.Parse(o.selector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as label selector")
		}
	}

	fieldSelector := fields.Everything()
	if o.fieldSelector != "" {
		fieldSelector, err = fields.ParseSelector(o.fieldSelector)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector as field selector")
		}
	}

	var tailLines *int64
	if o.tail != -1 {
		tailLines = &o.tail
	}

	switch o.color {
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
	case "auto":
	default:
		return nil, errors.New("color should be one of 'always', 'never', or 'auto'")
	}

	template, err := o.generateTemplate()
	if err != nil {
		return nil, err
	}

	namespaces := makeUnique(o.namespaces)

	// --timezone
	location, err := time.LoadLocation(o.timezone)
	if err != nil {
		return nil, err
	}

	maxLogRequests := o.maxLogRequests
	if maxLogRequests == -1 {
		if o.noFollow {
			maxLogRequests = 5
		} else {
			maxLogRequests = 50
		}
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
		Follow:                !o.noFollow,
		Resource:              o.resource,
		OnlyLogLines:          o.onlyLogLines,
		MaxLogRequests:        maxLogRequests,

		Out:    o.Out,
		ErrOut: o.ErrOut,
	}, nil
}

// setVerbosity sets the log level verbosity
func (o *options) setVerbosity() error {
	if o.verbosity != 0 {
		// klog does not have an external method to set verbosity,
		// so we need to set it by a flag.
		// See https://github.com/kubernetes/klog/issues/336 for details
		var fs goflag.FlagSet
		klog.InitFlags(&fs)
		return fs.Set("v", strconv.Itoa(o.verbosity))
	}
	return nil
}

// AddFlags adds all the flags used by stern.
func (o *options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.allNamespaces, "all-namespaces", "A", o.allNamespaces, "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.")
	fs.StringVar(&o.color, "color", o.color, "Force set color output. 'auto':  colorize if tty attached, 'always': always colorize, 'never': never colorize.")
	fs.StringVar(&o.completion, "completion", o.completion, "Output stern command-line completion code for the specified shell. Can be 'bash', 'zsh' or 'fish'.")
	fs.StringVarP(&o.container, "container", "c", o.container, "Container name when multiple containers in pod. (regular expression)")
	fs.StringSliceVar(&o.containerStates, "container-state", o.containerStates, "Tail containers with state in running, waiting, terminated, or all. 'all' matches all container states. To specify multiple states, repeat this or set comma-separated value.")
	fs.StringVar(&o.context, "context", o.context, "Kubernetes context to use. Default to current context configured in kubeconfig.")
	fs.StringArrayVarP(&o.exclude, "exclude", "e", o.exclude, "Log lines to exclude. (regular expression)")
	fs.StringArrayVarP(&o.excludeContainer, "exclude-container", "E", o.excludeContainer, "Container name to exclude when multiple containers in pod. (regular expression)")
	fs.StringArrayVar(&o.excludePod, "exclude-pod", o.excludePod, "Pod name to exclude. (regular expression)")
	fs.BoolVar(&o.noFollow, "no-follow", o.noFollow, "Exit when all logs have been shown.")
	fs.StringArrayVarP(&o.include, "include", "i", o.include, "Log lines to include. (regular expression)")
	fs.BoolVar(&o.initContainers, "init-containers", o.initContainers, "Include or exclude init containers.")
	fs.BoolVar(&o.ephemeralContainers, "ephemeral-containers", o.ephemeralContainers, "Include or exclude ephemeral containers.")
	fs.StringVar(&o.kubeConfig, "kubeconfig", o.kubeConfig, "Path to kubeconfig file to use. Default to KUBECONFIG variable then ~/.kube/config path.")
	fs.StringVar(&o.kubeConfig, "kube-config", o.kubeConfig, "Path to kubeconfig file to use.")
	_ = fs.MarkDeprecated("kube-config", "Use --kubeconfig instead.")
	fs.StringSliceVarP(&o.namespaces, "namespace", "n", o.namespaces, "Kubernetes namespace to use. Default to namespace configured in kubernetes context. To specify multiple namespaces, repeat this or set comma-separated value.")
	fs.IntVar(&o.maxLogRequests, "max-log-requests", o.maxLogRequests, "Maximum number of concurrent logs to request. Defaults to 50, but 5 when specifying --no-follow")
	fs.StringVarP(&o.output, "output", "o", o.output, "Specify predefined template. Currently support: [default, raw, json, extjson, ppextjson]")
	fs.BoolVarP(&o.prompt, "prompt", "p", o.prompt, "Toggle interactive prompt for selecting 'app.kubernetes.io/instance' label values.")
	fs.StringVarP(&o.selector, "selector", "l", o.selector, "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.StringVar(&o.fieldSelector, "field-selector", o.fieldSelector, "Selector (field query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.DurationVarP(&o.since, "since", "s", o.since, "Return logs newer than a relative duration like 5s, 2m, or 3h.")
	fs.Int64Var(&o.tail, "tail", o.tail, "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.")
	fs.StringVar(&o.template, "template", o.template, "Template to use for log lines, leave empty to use --output flag.")
	fs.StringVarP(&o.templateFile, "template-file", "T", o.templateFile, "Path to template to use for log lines, leave empty to use --output flag.")
	fs.BoolVarP(&o.timestamps, "timestamps", "t", o.timestamps, "Print timestamps.")
	fs.StringVar(&o.timezone, "timezone", o.timezone, "Set timestamps to specific timezone.")
	fs.BoolVar(&o.onlyLogLines, "only-log-lines", o.onlyLogLines, "Print only log lines")
	fs.IntVar(&o.verbosity, "verbosity", o.verbosity, "Number of the log level verbosity")
	fs.BoolVarP(&o.version, "version", "v", o.version, "Print the version and exit.")
}

func (o *options) generateTemplate() (*template.Template, error) {
	t := o.template
	if o.templateFile != "" {
		data, err := os.ReadFile(o.templateFile)
		if err != nil {
			return nil, err
		}
		t = string(data)
	}
	if t == "" {
		switch o.output {
		case "default":
			t = "{{color .PodColor .PodName}} {{color .ContainerColor .ContainerName}} {{.Message}}"
			if o.allNamespaces || len(o.namespaces) > 1 {
				t = fmt.Sprintf("{{color .PodColor .Namespace}} %s", t)
			}
		case "raw":
			t = "{{.Message}}"
		case "json":
			t = "{{json .}}"
		case "extjson":
			t = "\"pod\": \"{{color .PodColor .PodName}}\", \"container\": \"{{color .ContainerColor .ContainerName}}\", \"message\": {{extjson .Message}}"
			if o.allNamespaces {
				t = fmt.Sprintf("\"namespace\": \"{{color .PodColor .Namespace}}\", %s", t)
			}
			t = fmt.Sprintf("{%s}", t)
		case "ppextjson":
			t = "  \"pod\": \"{{color .PodColor .PodName}}\",\n  \"container\": \"{{color .ContainerColor .ContainerName}}\",\n  \"message\": {{extjson .Message}}"
			if o.allNamespaces {
				t = fmt.Sprintf("  \"namespace\": \"{{color .PodColor .Namespace}}\",\n%s", t)
			}
			t = fmt.Sprintf("{\n%s\n}", t)
		default:
			return nil, errors.New("output should be one of 'default', 'raw', 'json', 'extjson', and 'ppextjson'")
		}
	}
	if !strings.HasSuffix(t, "\n") {
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
		"tryParseJSON": func(text string) map[string]interface{} {
			obj := make(map[string]interface{})
			if err := json.Unmarshal([]byte(text), &obj); err != nil {
				return nil
			}
			return obj
		},
		"parseJSON": func(text string) (map[string]interface{}, error) {
			obj := make(map[string]interface{})
			if err := json.Unmarshal([]byte(text), &obj); err != nil {
				return obj, err
			}
			return obj, nil
		},
		"extjson": func(in string) (string, error) {
			if json.Valid([]byte(in)) {
				return strings.TrimSuffix(in, "\n"), nil
			}
			b, err := json.Marshal(in)
			if err != nil {
				return "", err
			}
			return strings.TrimSuffix(string(b), "\n"), nil
		},
		"formatTsRFC3339Nano": func(ts any) string {
			return cast.ToTime(ts).Format(time.RFC3339Nano)
		},
		"color": func(color color.Color, text string) string {
			return color.SprintFunc()(text)
		},
		"colorBlack":   color.BlackString,
		"colorRed":     color.RedString,
		"colorGreen":   color.GreenString,
		"colorYellow":  color.YellowString,
		"colorBlue":    color.BlueString,
		"colorMagenta": color.MagentaString,
		"colorCyan":    color.CyanString,
		"colorWhite":   color.WhiteString,
		"levelColor": func(level string) string {
			var levelColor *color.Color
			switch strings.ToLower(level) {
			case "debug":
				levelColor = color.New(color.FgMagenta)
			case "info":
				levelColor = color.New(color.FgBlue)
			case "warn":
				levelColor = color.New(color.FgYellow)
			case "error":
				levelColor = color.New(color.FgRed)
			case "dpanic":
				levelColor = color.New(color.FgRed)
			case "panic":
				levelColor = color.New(color.FgRed)
			case "fatal":
				levelColor = color.New(color.FgCyan)
			default:
			}
			return levelColor.SprintFunc()(level)
		},
	}
	template, err := template.New("log").Funcs(funs).Parse(t)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse template")
	}
	return template, err
}

func NewSternCmd(stream genericclioptions.IOStreams) (*cobra.Command, error) {
	o := NewOptions(stream)

	cmd := &cobra.Command{
		Use:   "stern pod-query",
		Short: "Tail multiple pods and containers from Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Output version information and exit
			if o.version {
				outputVersionInfo(o.Out)
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
		ValidArgsFunction: queryCompletionFunc(o),
	}

	o.AddFlags(cmd.Flags())

	if err := registerCompletionFuncForFlags(cmd, o); err != nil {
		return cmd, err
	}

	return cmd, nil
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

func compileREs(exprs []string) ([]*regexp.Regexp, error) {
	var regexps []*regexp.Regexp
	for _, s := range exprs {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		regexps = append(regexps, re)
	}
	return regexps, nil
}
