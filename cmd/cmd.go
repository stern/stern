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
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stern/stern/stern"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	// load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Use "~" to avoid exposing the user name in the help message
var defaultConfigFilePath = "~/.config/stern/config.yaml"

type options struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams

	excludePod          []string
	container           string
	excludeContainer    []string
	containerStates     []string
	timestamps          string
	timezone            string
	since               time.Duration
	namespaces          []string
	condition           string
	exclude             []string
	include             []string
	highlight           []string
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
	node                string
	configFilePath      string
	showHiddenOptions   bool
	stdin               bool
	diffContainer       bool
	podColors           []string
	containerColors     []string

	client       kubernetes.Interface
	clientConfig clientcmd.ClientConfig
}

func NewOptions(streams genericclioptions.IOStreams) *options {
	configFlags := genericclioptions.NewConfigFlags(true)
	// stern has its own namespace flag, so disable the one in configFlags
	configFlags.Namespace = nil

	return &options{
		configFlags: configFlags,
		IOStreams:   streams,

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
		timestamps:          "",
		timezone:            "Local",
		prompt:              false,
		noFollow:            false,
		maxLogRequests:      -1,
		configFilePath:      defaultConfigFilePath,
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

	envVar, ok := os.LookupEnv("STERNCONFIG")
	if ok {
		o.configFilePath = envVar
	}

	o.clientConfig = o.configFlags.ToRawKubeConfigLoader()

	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	o.client = kubernetes.NewForConfigOrDie(restConfig)

	if len(o.namespaces) == 0 {
		namespace, _, err := o.clientConfig.Namespace()
		if err != nil {
			return err
		}
		o.namespaces = []string{namespace}
	}

	return nil
}

func (o *options) Validate() error {
	if !o.prompt && o.podQuery == "" && o.resource == "" && o.selector == "" && o.fieldSelector == "" && !o.stdin {
		return errors.New("One of pod-query, --selector, --field-selector, --prompt or --stdin is required")
	}
	if o.selector != "" && o.resource != "" {
		return errors.New("--selector and the <resource>/<name> query cannot be set at the same time")
	}
	if o.noFollow && o.tail == 0 {
		return errors.New("--no-follow cannot be used with --tail=0")
	}

	return nil
}

func (o *options) Run(cmd *cobra.Command) error {
	if err := o.setColorList(); err != nil {
		return err
	}

	config, err := o.sternConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if o.prompt {
		if err := promptHandler(ctx, o.client, config, o.Out); err != nil {
			return err
		}
	}

	return stern.Run(ctx, o.client, config)
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

	highlight, err := compileREs(o.highlight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regular expression for highlight filter")
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

	fieldSelector, err := o.generateFieldSelector()
	if err != nil {
		return nil, err
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

	var timestampFormat string
	switch o.timestamps {
	case "default":
		timestampFormat = stern.TimestampFormatDefault
	case "short":
		timestampFormat = stern.TimestampFormatShort
	case "":
	default:
		return nil, errors.New("timestamps should be one of 'default', or 'short'")
	}

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
		Namespaces:            namespaces,
		PodQuery:              pod,
		ExcludePodQuery:       excludePod,
		Timestamps:            timestampFormat != "",
		TimestampFormat:       timestampFormat,
		Location:              location,
		ContainerQuery:        container,
		ExcludeContainerQuery: excludeContainer,
		Condition:             o.condition,
		ContainerStates:       containerStates,
		Exclude:               exclude,
		Include:               include,
		Highlight:             highlight,
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
		Stdin:                 o.stdin,
		DiffContainer:         o.diffContainer,

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

func (o *options) setColorList() error {
	if len(o.podColors) > 0 || len(o.containerColors) > 0 {
		return stern.SetColorList(o.podColors, o.containerColors)
	}
	return nil
}

// overrideFlagSetDefaultFromConfig overrides the default value of the flagSets
// from the config file
func (o *options) overrideFlagSetDefaultFromConfig(fs *pflag.FlagSet) error {
	expanded, err := homedir.Expand(o.configFilePath)
	if err != nil {
		return err
	}

	if o.configFilePath == defaultConfigFilePath {
		if _, err := os.Stat(expanded); os.IsNotExist(err) {
			return nil
		}
	}

	configFile, err := os.Open(expanded)
	if err != nil {
		return err
	}

	data := make(map[string]interface{})

	if err := yaml.NewDecoder(configFile).Decode(data); err != nil && err != io.EOF {
		return err
	}

	for name, value := range data {
		flag := fs.Lookup(name)
		if flag == nil {
			// To avoid command execution failure, we only output a warning
			// message instead of exiting with an error if an unknown option is
			// specified.
			klog.Warningf("Unknown option specified in the config file: %s", name)
			continue
		}

		// flag has higher priority than the config file
		if flag.Changed {
			continue
		}

		if valueSlice, ok := value.([]any); ok {
			// the value is an array
			if flagSlice, ok := flag.Value.(pflag.SliceValue); ok {
				values := make([]string, len(valueSlice))
				for i, v := range valueSlice {
					values[i] = fmt.Sprint(v)
				}
				if err := flagSlice.Replace(values); err != nil {
					return fmt.Errorf("invalid value %q for %q in the config file: %v", value, name, err)
				}
				continue
			}
		}

		if err := flag.Value.Set(fmt.Sprint(value)); err != nil {
			return fmt.Errorf("invalid value %q for %q in the config file: %v", value, name, err)
		}
	}

	return nil
}

// AddFlags adds all the flags used by stern.
func (o *options) AddFlags(fs *pflag.FlagSet) {
	o.addKubernetesFlags(fs)

	fs.BoolVarP(&o.allNamespaces, "all-namespaces", "A", o.allNamespaces, "If present, tail across all namespaces. A specific namespace is ignored even if specified with --namespace.")
	fs.StringVar(&o.color, "color", o.color, "Force set color output. 'auto':  colorize if tty attached, 'always': always colorize, 'never': never colorize.")
	fs.StringVar(&o.completion, "completion", o.completion, "Output stern command-line completion code for the specified shell. Can be 'bash', 'zsh' or 'fish'.")
	fs.StringVarP(&o.container, "container", "c", o.container, "Container name when multiple containers in pod. (regular expression)")
	fs.StringSliceVar(&o.containerStates, "container-state", o.containerStates, "Tail containers with state in running, waiting, terminated, or all. 'all' matches all container states. To specify multiple states, repeat this or set comma-separated value.")
	fs.StringArrayVarP(&o.exclude, "exclude", "e", o.exclude, "Log lines to exclude. (regular expression)")
	fs.StringArrayVarP(&o.excludeContainer, "exclude-container", "E", o.excludeContainer, "Container name to exclude when multiple containers in pod. (regular expression)")
	fs.StringArrayVar(&o.excludePod, "exclude-pod", o.excludePod, "Pod name to exclude. (regular expression)")
	fs.StringVar(&o.condition, "condition", o.condition, "The condition to filter on: [condition-name[=condition-value]. The default condition-value is true. Match is case-insensitive.")
	fs.BoolVar(&o.noFollow, "no-follow", o.noFollow, "Exit when all logs have been shown.")
	fs.StringArrayVarP(&o.include, "include", "i", o.include, "Log lines to include. (regular expression)")
	fs.StringArrayVarP(&o.highlight, "highlight", "H", o.highlight, "Log lines to highlight. (regular expression)")
	fs.BoolVar(&o.initContainers, "init-containers", o.initContainers, "Include or exclude init containers.")
	fs.BoolVar(&o.ephemeralContainers, "ephemeral-containers", o.ephemeralContainers, "Include or exclude ephemeral containers.")
	fs.StringSliceVarP(&o.namespaces, "namespace", "n", o.namespaces, "Kubernetes namespace to use. Default to namespace configured in kubernetes context. To specify multiple namespaces, repeat this or set comma-separated value.")
	fs.StringVar(&o.node, "node", o.node, "Node name to filter on.")
	fs.IntVar(&o.maxLogRequests, "max-log-requests", o.maxLogRequests, "Maximum number of concurrent logs to request. Defaults to 50, but 5 when specifying --no-follow")
	fs.StringVarP(&o.output, "output", "o", o.output, "Specify predefined template. Currently support: [default, raw, json, extjson, ppextjson]")
	fs.BoolVarP(&o.prompt, "prompt", "p", o.prompt, "Toggle interactive prompt for selecting 'app.kubernetes.io/instance' label values.")
	fs.StringVarP(&o.selector, "selector", "l", o.selector, "Selector (label query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.StringVar(&o.fieldSelector, "field-selector", o.fieldSelector, "Selector (field query) to filter on. If present, default to \".*\" for the pod-query.")
	fs.DurationVarP(&o.since, "since", "s", o.since, "Return logs newer than a relative duration like 5s, 2m, or 3h.")
	fs.Int64Var(&o.tail, "tail", o.tail, "The number of lines from the end of the logs to show. Defaults to -1, showing all logs.")
	fs.StringVar(&o.template, "template", o.template, "Template to use for log lines, leave empty to use --output flag.")
	fs.StringVarP(&o.templateFile, "template-file", "T", o.templateFile, "Path to template to use for log lines, leave empty to use --output flag. It overrides --template option.")
	fs.StringVarP(&o.timestamps, "timestamps", "t", o.timestamps, "Print timestamps with the specified format. One of 'default' or 'short' in the form '--timestamps=format' ('=' cannot be omitted). If specified but without value, 'default' is used.")
	fs.StringVar(&o.timezone, "timezone", o.timezone, "Set timestamps to specific timezone.")
	fs.BoolVar(&o.onlyLogLines, "only-log-lines", o.onlyLogLines, "Print only log lines")
	fs.StringVar(&o.configFilePath, "config", o.configFilePath, "Path to the stern config file")
	fs.IntVar(&o.verbosity, "verbosity", o.verbosity, "Number of the log level verbosity")
	fs.BoolVarP(&o.version, "version", "v", o.version, "Print the version and exit.")
	fs.BoolVar(&o.showHiddenOptions, "show-hidden-options", o.showHiddenOptions, "Print a list of hidden options.")
	fs.BoolVar(&o.stdin, "stdin", o.stdin, "Parse logs from stdin. All Kubernetes related flags are ignored when it is set.")
	fs.BoolVarP(&o.diffContainer, "diff-container", "d", o.diffContainer, "Display different colors for different containers.")
	fs.StringSliceVar(&o.podColors, "pod-colors", o.podColors, "Specifies the colors used to highlight pod names. Provide colors as a comma-separated list using SGR (Select Graphic Rendition) sequences, e.g., \"91,92,93,94,95,96\".")
	fs.StringSliceVar(&o.containerColors, "container-colors", o.containerColors, "Specifies the colors used to highlight container names. Use the same format as --pod-colors. Defaults to the values of --pod-colors if omitted, and must match its length.")

	fs.Lookup("timestamps").NoOptDefVal = "default"
}

func (o *options) addKubernetesFlags(fs *pflag.FlagSet) {
	flagset := pflag.NewFlagSet("", pflag.ExitOnError)
	o.configFlags.AddFlags(flagset)
	flagset.VisitAll(func(f *pflag.Flag) {
		// Hide Kubernetes flags except some
		if !(f.Name == "kubeconfig" || f.Name == "context") {
			f.Hidden = true
		}

		// `server` flag in configFlags has `s` shorthand, which is used by stern
		// as shorthand for `since` flag, so do not use it.
		if f.Name == "server" {
			f.Shorthand = ""
		}
	})
	fs.AddFlagSet(flagset)
}

func (o *options) outputHiddenOptions() {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	o.AddFlags(fs)
	fs.VisitAll(func(f *pflag.Flag) {
		f.Hidden = !f.Hidden
	})
	fmt.Println("The following options can also be used in stern:")
	fs.PrintDefaults()
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
			decoder := json.NewDecoder(strings.NewReader(text))
			decoder.UseNumber()
			obj := make(map[string]interface{})
			if err := decoder.Decode(&obj); err != nil {
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
		"extractJSONParts": func(text string, part ...string) (string, error) {
			obj := make(map[string]interface{})
			if err := json.Unmarshal([]byte(text), &obj); err != nil {
				return "", err
			}
			parts := make([]string, 0)
			for _, key := range part {
				parts = append(parts, fmt.Sprintf("%v", obj[key]))
			}
			return strings.Join(parts, ", "), nil
		},
		"tryExtractJSONParts": func(text string, part ...string) string {
			obj := make(map[string]interface{})
			if err := json.Unmarshal([]byte(text), &obj); err != nil {
				return text
			}
			parts := make([]string, 0)
			for _, key := range part {
				parts = append(parts, fmt.Sprintf("%v", obj[key]))
			}
			return strings.Join(parts, ", ")
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
		"prettyJSON": func(value any) string {
			var data map[string]any

			switch v := value.(type) {
			case string:
				if err := json.Unmarshal([]byte(v), &data); err != nil {
					return v
				}
			case map[string]any:
				data = v
			default:
				return fmt.Sprintf("%v", value)
			}

			b, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", value)
			}

			return string(b)
		},
		"toRFC3339Nano": func(ts any) string {
			return toTime(ts).Format(time.RFC3339Nano)
		},
		"toUTC": func(ts any) time.Time {
			return toTime(ts).UTC()
		},
		"toTimestamp": func(ts any, layout string, optionalTZ ...string) (string, error) {
			t, parseErr := toTimeE(ts)
			if parseErr != nil {
				return "", parseErr
			}

			var tz string
			if len(optionalTZ) > 0 {
				tz = optionalTZ[0]
			}

			loc, loadErr := time.LoadLocation(tz)
			if loadErr != nil {
				return "", loadErr
			}

			return t.In(loc).Format(layout), nil
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
		"levelColor": func(value any) string {
			switch level := value.(type) {
			case string:
				var levelColor *color.Color
				switch strings.ToLower(level) {
				case "debug":
					levelColor = color.New(color.FgMagenta)
				case "info":
					levelColor = color.New(color.FgBlue)
				case "warn":
					levelColor = color.New(color.FgYellow)
				case "warning":
					levelColor = color.New(color.FgYellow)
				case "error":
					levelColor = color.New(color.FgRed)
				case "dpanic":
					levelColor = color.New(color.FgRed)
				case "panic":
					levelColor = color.New(color.FgRed)
				case "fatal":
					levelColor = color.New(color.FgCyan)
				case "critical":
					levelColor = color.New(color.FgCyan)
				default:
					return level
				}
				return levelColor.SprintFunc()(level)
			default:
				return ""
			}
		},
		"bunyanLevelColor": func(value any) string {
			var lv int64
			var err error

			switch level := value.(type) {
			// tryParseJSON yields json.Number
			case json.Number:
				lv, err = level.Int64()
				if err != nil {
					return ""
				}
			// parseJSON yields float64
			case float64:
				lv = int64(level)
			default:
				return ""
			}

			var levelColor *color.Color
			switch {
			case lv < 30:
				levelColor = color.New(color.FgMagenta)
			case lv < 40:
				levelColor = color.New(color.FgBlue)
			case lv < 50:
				levelColor = color.New(color.FgYellow)
			case lv < 60:
				levelColor = color.New(color.FgRed)
			case lv < 100:
				levelColor = color.New(color.FgCyan)
			default:
				return strconv.FormatInt(lv, 10)
			}
			return levelColor.SprintFunc()(lv)
		},
	}
	template, err := template.New("log").Funcs(funs).Parse(t)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse template")
	}
	return template, err
}

func (o *options) generateFieldSelector() (fields.Selector, error) {
	var queries []string
	if o.fieldSelector != "" {
		queries = append(queries, o.fieldSelector)
	}
	if o.node != "" {
		queries = append(queries, fmt.Sprintf("spec.nodeName=%s", o.node))
	}
	if len(queries) == 0 {
		return fields.Everything(), nil
	}

	fieldSelector, err := fields.ParseSelector(strings.Join(queries, ","))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse selector as field selector")
	}
	return fieldSelector, nil
}

func NewSternCmd(stream genericclioptions.IOStreams) (*cobra.Command, error) {
	o := NewOptions(stream)

	cmd := &cobra.Command{
		Use:   "stern pod-query",
		Short: "Tail multiple pods and containers from Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			// klog's v flag should be initialized before creating a k8s client
			if err := o.setVerbosity(); err != nil {
				return err
			}

			// Output version information and exit
			if o.version {
				outputVersionInfo(o.Out)
				return nil
			}

			// Output shell completion code for the specified shell and exit
			if o.completion != "" {
				return runCompletion(o.completion, cmd, o.Out)
			}

			if o.showHiddenOptions {
				o.outputHiddenOptions()
				return nil
			}

			if err := o.Complete(args); err != nil {
				return err
			}

			if err := o.overrideFlagSetDefaultFromConfig(cmd.Flags()); err != nil {
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

	cmd.SetUsageTemplate(cmd.UsageTemplate() + "\nUse \"stern --show-hidden-options\" for a list of hidden command-line options.\n")

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
