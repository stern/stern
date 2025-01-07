package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

func TestSternCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		out  string
	}{
		{
			"Output version info with --version",
			[]string{"--version"},
			"version: dev",
		},
		{
			"Output completion code for bash with --completion=bash",
			[]string{"--completion=bash"},
			"complete -o default -F __start_stern stern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, _, out, _ := genericclioptions.NewTestIOStreams()
			stern, err := NewSternCmd(streams)
			if err != nil {
				t.Fatal(err)
			}
			stern.SetArgs(tt.args)

			if err := stern.Execute(); err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(out.String(), tt.out) {
				t.Errorf("expected to contain %s, but actual %s", tt.out, out.String())
			}
		})
	}
}

func TestOptionsComplete(t *testing.T) {
	streams := genericclioptions.NewTestIOStreamsDiscard()

	tests := []struct {
		name                   string
		env                    map[string]string
		args                   []string
		expectedConfigFilePath string
	}{
		{
			name:                   "No environment variables",
			env:                    map[string]string{},
			args:                   []string{},
			expectedConfigFilePath: defaultConfigFilePath,
		},
		{
			name: "Set STERNCONFIG env to ./config.yaml",
			env: map[string]string{
				"STERNCONFIG": "./config.yaml",
			},
			args:                   []string{},
			expectedConfigFilePath: "./config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			o := NewOptions(streams)
			_ = o.Complete(tt.args)

			if tt.expectedConfigFilePath != o.configFilePath {
				t.Errorf("expected %s for configFilePath, but got %s", tt.expectedConfigFilePath, o.configFilePath)
			}
		})
	}
}

func TestOptionsValidate(t *testing.T) {
	streams := genericclioptions.NewTestIOStreamsDiscard()

	tests := []struct {
		name string
		o    *options
		err  string
	}{
		{
			"No required options",
			NewOptions(streams),
			"One of pod-query, --selector, --field-selector, --prompt or --stdin is required",
		},
		{
			"Specify both selector and resource",
			func() *options {
				o := NewOptions(streams)
				o.selector = "app=nginx"
				o.resource = "deployment/nginx"

				return o
			}(),
			"--selector and the <resource>/<name> query cannot be set at the same time",
		},
		{
			"Specify both --no-follow and --tail=0",
			func() *options {
				o := NewOptions(streams)
				o.podQuery = "."
				o.noFollow = true
				o.tail = 0

				return o
			}(),
			"--no-follow cannot be used with --tail=0",
		},
		{
			"Use prompt",
			func() *options {
				o := NewOptions(streams)
				o.prompt = true

				return o
			}(),
			"",
		},
		{
			"Specify pod-query",
			func() *options {
				o := NewOptions(streams)
				o.podQuery = "."

				return o
			}(),
			"",
		},
		{
			"Specify selector",
			func() *options {
				o := NewOptions(streams)
				o.selector = "app=nginx"

				return o
			}(),
			"",
		},
		{
			"Specify fieldSelector",
			func() *options {
				o := NewOptions(streams)
				o.fieldSelector = "spec.nodeName=kind-kind"

				return o
			}(),
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.o.Validate()
			if err == nil {
				if tt.err != "" {
					t.Errorf("expected %q err, but actual no err", tt.err)
				}
			} else {
				if tt.err != err.Error() {
					t.Errorf("expected %q err, but actual %q", tt.err, err)
				}
			}
		})
	}
}

func TestOptionsGenerateTemplate(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	streams := genericclioptions.NewTestIOStreamsDiscard()

	tests := []struct {
		name      string
		o         *options
		message   string
		want      string
		wantError bool
	}{
		{
			"output=default",
			func() *options {
				o := NewOptions(streams)
				o.output = "default"

				return o
			}(),
			"default message",
			"pod1 container1 default message\n",
			false,
		},
		{
			"output=default+allNamespaces",
			func() *options {
				o := NewOptions(streams)
				o.output = "default"
				o.allNamespaces = true

				return o
			}(),
			"default message",
			"ns1 pod1 container1 default message\n",
			false,
		},
		{
			"output=raw",
			func() *options {
				o := NewOptions(streams)
				o.output = "raw"

				return o
			}(),
			"raw message",
			"raw message\n",
			false,
		},
		{
			"output=json",
			func() *options {
				o := NewOptions(streams)
				o.output = "json"

				return o
			}(),
			"json message",
			`{"message":"json message","nodeName":"node1","namespace":"ns1","podName":"pod1","containerName":"container1"}
`,
			false,
		},
		{
			"output=extjson",
			func() *options {
				o := NewOptions(streams)
				o.output = "extjson"

				return o
			}(),
			`{"msg":"extjson message"}`,
			`{"pod": "pod1", "container": "container1", "message": {"msg":"extjson message"}}
`,
			false,
		},
		{
			"output=extjson+allNamespaces",
			func() *options {
				o := NewOptions(streams)
				o.output = "extjson"
				o.allNamespaces = true

				return o
			}(),
			`{"msg":"extjson message"}`,
			`{"namespace": "ns1", "pod": "pod1", "container": "container1", "message": {"msg":"extjson message"}}
`,
			false,
		},
		{
			"output=ppextjson",
			func() *options {
				o := NewOptions(streams)
				o.output = "ppextjson"

				return o
			}(),
			`{"msg":"ppextjson message"}`,
			`{
  "pod": "pod1",
  "container": "container1",
  "message": {"msg":"ppextjson message"}
}
`,
			false,
		},
		{
			"output=ppextjson+allNamespaces",
			func() *options {
				o := NewOptions(streams)
				o.output = "ppextjson"
				o.allNamespaces = true

				return o
			}(),
			`{"msg":"ppextjson message"}`,
			`{
  "namespace": "ns1",
  "pod": "pod1",
  "container": "container1",
  "message": {"msg":"ppextjson message"}
}
`,
			false,
		},
		{
			"invalid output",
			func() *options {
				o := NewOptions(streams)
				o.output = "invalid"

				return o
			}(),
			"message",
			"",
			true,
		},
		{
			"template",
			func() *options {
				o := NewOptions(streams)
				o.template = "Message={{.Message}} NodeName={{.NodeName}} Namespace={{.Namespace}} PodName={{.PodName}} ContainerName={{.ContainerName}}"

				return o
			}(),
			"template message", // no new line
			"Message=template message NodeName=node1 Namespace=ns1 PodName=pod1 ContainerName=container1",
			false,
		},
		{
			"invalid template",
			func() *options {
				o := NewOptions(streams)
				o.template = "{{invalid"

				return o
			}(),
			"template message",
			"",
			true,
		},
		{
			"template-file",
			func() *options {
				o := NewOptions(streams)
				o.templateFile = "test.tpl"

				return o
			}(),
			"template message",
			"pod1 container1 template message",
			false,
		},
		{
			"template-file-json-log-ts-float",
			func() *options {
				o := NewOptions(streams)
				o.templateFile = "test.tpl"

				return o
			}(),
			`{"ts": 123, "level": "INFO", "msg": "template message"}`,
			"pod1 container1 [1970-01-01T00:02:03Z] INFO template message",
			false,
		},
		{
			"template-file-json-log-ts-str",
			func() *options {
				o := NewOptions(streams)
				o.templateFile = "test.tpl"

				return o
			}(),
			`{"ts": "1970-01-01T01:02:03+01:00", "level": "INFO", "msg": "template message"}`,
			"pod1 container1 [1970-01-01T00:02:03Z] INFO template message",
			false,
		},
		{
			"template-to-timestamp-with-timezone",
			func() *options {
				o := NewOptions(streams)
				o.template = `{{ toTimestamp .Message "Jan 02 2006 15:04 MST" "US/Eastern" }}`
				return o
			}(),
			`2024-01-01T05:00:00`,
			`Jan 01 2024 00:00 EST`,
			false,
		},
		{
			"template-to-timestamp-without-timezone",
			func() *options {
				o := NewOptions(streams)
				o.template = `{{ toTimestamp .Message "Jan 02 2006 15:04 MST" }}`
				return o
			}(),
			`2024-01-01T05:00:00`,
			`Jan 01 2024 05:00 UTC`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := stern.Log{
				Message:        tt.message,
				NodeName:       "node1",
				Namespace:      "ns1",
				PodName:        "pod1",
				ContainerName:  "container1",
				PodColor:       color.New(color.FgRed),
				ContainerColor: color.New(color.FgBlue),
			}
			tmpl, err := tt.o.generateTemplate()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, but got no error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, log); err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if want, got := tt.want, buf.String(); want != got {
				t.Errorf("want %v, but got %v", want, got)
			}
		})
	}
}

func TestOptionsSternConfig(t *testing.T) {
	streams := genericclioptions.NewTestIOStreamsDiscard()

	local, _ := time.LoadLocation("Local")
	utc, _ := time.LoadLocation("UTC")
	labelSelector, _ := labels.Parse("l=sel")
	fieldSelector, _ := fields.ParseSelector("f=field,spec.nodeName=node1")

	re := regexp.MustCompile

	defaultConfig := func() *stern.Config {
		return &stern.Config{
			Namespaces:            []string{},
			PodQuery:              re(""),
			ExcludePodQuery:       nil,
			Timestamps:            false,
			TimestampFormat:       "",
			Location:              local,
			ContainerQuery:        re(".*"),
			ExcludeContainerQuery: nil,
			ContainerStates:       []stern.ContainerState{stern.ALL_STATES},
			Exclude:               nil,
			Include:               nil,
			Highlight:             nil,
			InitContainers:        true,
			EphemeralContainers:   true,
			Since:                 48 * time.Hour,
			AllNamespaces:         false,
			LabelSelector:         labels.Everything(),
			FieldSelector:         fields.Everything(),
			TailLines:             nil,
			Template:              nil, // ignore when comparing
			Follow:                true,
			Resource:              "",
			OnlyLogLines:          false,
			MaxLogRequests:        50,

			Out:    streams.Out,
			ErrOut: streams.ErrOut,
		}
	}

	tests := []struct {
		name      string
		o         *options
		want      *stern.Config
		wantError bool
	}{
		{
			"default",
			NewOptions(streams),
			defaultConfig(),
			false,
		},
		{
			"change all options",
			func() *options {
				o := NewOptions(streams)
				o.namespaces = []string{"ns1", "ns2"}
				o.podQuery = "query1"
				o.excludePod = []string{"exp1", "exp2"}
				o.timestamps = "default"
				o.timezone = "UTC" // Location
				o.container = "container1"
				o.excludeContainer = []string{"exc1", "exc2"}
				o.containerStates = []string{"running", "terminated"}
				o.exclude = []string{"ex1", "ex2"}
				o.include = []string{"in1", "in2"}
				o.highlight = []string{"hi1", "hi2"}
				o.initContainers = false
				o.ephemeralContainers = false
				o.since = 1 * time.Hour
				o.allNamespaces = true
				o.selector = "l=sel"
				o.fieldSelector = "f=field"
				o.tail = 10
				o.noFollow = true // Follow = false
				o.maxLogRequests = 30
				o.resource = "res1"
				o.onlyLogLines = true
				o.node = "node1"

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				c.Namespaces = []string{"ns1", "ns2"}
				c.PodQuery = re("query1")
				c.ExcludePodQuery = []*regexp.Regexp{re("exp1"), re("exp2")}
				c.Timestamps = true
				c.TimestampFormat = stern.TimestampFormatDefault
				c.Location = utc
				c.ContainerQuery = re("container1")
				c.ExcludeContainerQuery = []*regexp.Regexp{re("exc1"), re("exc2")}
				c.ContainerStates = []stern.ContainerState{stern.RUNNING, stern.TERMINATED}
				c.Exclude = []*regexp.Regexp{re("ex1"), re("ex2")}
				c.Include = []*regexp.Regexp{re("in1"), re("in2")}
				c.Highlight = []*regexp.Regexp{re("hi1"), re("hi2")}
				c.InitContainers = false
				c.EphemeralContainers = false
				c.Since = 1 * time.Hour
				c.AllNamespaces = true
				c.LabelSelector = labelSelector
				c.FieldSelector = fieldSelector
				c.TailLines = ptr.To[int64](10)
				c.Follow = false
				c.Resource = "res1"
				c.OnlyLogLines = true
				c.MaxLogRequests = 30

				return c
			}(),
			false,
		},
		{
			"fieldSelector without node",
			func() *options {
				o := NewOptions(streams)
				o.fieldSelector = "f=field"

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				sel, _ := fields.ParseSelector("f=field")
				c.FieldSelector = sel

				return c
			}(),
			false,
		},
		{
			"node without fieldSelector",
			func() *options {
				o := NewOptions(streams)
				o.node = "node1"

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				sel, _ := fields.ParseSelector("spec.nodeName=node1")
				c.FieldSelector = sel

				return c
			}(),
			false,
		},
		{
			"timestamp=short",
			func() *options {
				o := NewOptions(streams)
				o.timestamps = "short"

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				c.Timestamps = true
				c.TimestampFormat = stern.TimestampFormatShort

				return c
			}(),
			false,
		},
		{
			"noFollow has the different default",
			func() *options {
				o := NewOptions(streams)
				o.noFollow = true // Follow = false

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				c.Follow = false
				c.MaxLogRequests = 5 // default of noFollow

				return c
			}(),
			false,
		},
		{
			"nil should be allowed",
			func() *options {
				o := NewOptions(streams)
				o.excludePod = nil
				o.excludeContainer = nil
				o.containerStates = nil
				o.namespaces = nil
				o.exclude = nil
				o.include = nil
				o.highlight = nil

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				c.ContainerStates = []stern.ContainerState{}

				return c
			}(),
			false,
		},
		{
			"error podQuery",
			func() *options {
				o := NewOptions(streams)
				o.podQuery = "[invalid"

				return o
			}(),
			nil,
			true,
		},
		{
			"error excludePod",
			func() *options {
				o := NewOptions(streams)
				o.excludePod = []string{"exp1", "[invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error container",
			func() *options {
				o := NewOptions(streams)
				o.container = "[invalid"

				return o
			}(),
			nil,
			true,
		},
		{
			"error excludeContainer",
			func() *options {
				o := NewOptions(streams)
				o.excludeContainer = []string{"exc1", "[invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error exclude",
			func() *options {
				o := NewOptions(streams)
				o.exclude = []string{"ex1", "[invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error include",
			func() *options {
				o := NewOptions(streams)
				o.include = []string{"in1", "[invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error highlight",
			func() *options {
				o := NewOptions(streams)
				o.highlight = []string{"hi1", "[invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error containerStates",
			func() *options {
				o := NewOptions(streams)
				o.containerStates = []string{"running", "invalid"}

				return o
			}(),
			nil,
			true,
		},
		{
			"error selector",
			func() *options {
				o := NewOptions(streams)
				o.selector = "-"

				return o
			}(),
			nil,
			true,
		},
		{
			"error fieldSelector",
			func() *options {
				o := NewOptions(streams)
				o.fieldSelector = "-"

				return o
			}(),
			nil,
			true,
		},
		{
			"error color",
			func() *options {
				o := NewOptions(streams)
				o.color = "invalid"

				return o
			}(),
			nil,
			true,
		},
		{
			"error output",
			func() *options {
				o := NewOptions(streams)
				o.output = "invalid"

				return o
			}(),
			nil,
			true,
		},
		{
			"error timezone",
			func() *options {
				o := NewOptions(streams)
				o.timezone = "invalid"

				return o
			}(),
			nil,
			true,
		},
		{
			"error timestamps",
			func() *options {
				o := NewOptions(streams)
				o.timestamps = "invalid"

				return o
			}(),
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.sternConfig()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, but got no error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// We skip the template as it is difficult to check
			// and is tested in TestOptionsGenerateTemplate().
			got.Template = nil

			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("want %+v, but got %+v", tt.want, got)
			}
		})
	}
}

func TestOptionsOverrideFlagSetDefaultFromConfig(t *testing.T) {
	orig := defaultConfigFilePath
	defer func() {
		defaultConfigFilePath = orig
	}()

	defaultConfigFilePath = "./config.yaml"
	wd, _ := os.Getwd()

	tests := []struct {
		name                    string
		flagConfigFilePathValue string
		flagTailValue           string
		expectedTailValue       int64
		wantErr                 bool
	}{
		{
			name:                    "--config=testdata/config-tail1.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-tail1.yaml"),
			expectedTailValue:       1,
			wantErr:                 false,
		},
		{
			name:                    "--config=testdata/config-empty.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-empty.yaml"),
			expectedTailValue:       -1,
			wantErr:                 false,
		},
		{
			name:                    "--config=config-not-exist.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "config-not-exist.yaml"),
			wantErr:                 true,
		},
		{
			name:                    "--config=config-invalid.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-invalid.yaml"),
			wantErr:                 true,
		},
		{
			name:                    "--config=config-unknown-option.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-unknown-option.yaml"),
			expectedTailValue:       1,
			wantErr:                 false,
		},
		{
			name:                    "--config=config-tail-invalid-value.yaml",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-tail-invalid-value.yaml"),
			wantErr:                 true,
		},
		{
			name:              "config file path is not specified and config file does not exist",
			expectedTailValue: -1,
			wantErr:           false,
		},
		{
			name:                    "--config=testdata/config-tail1.yaml and --tail=2",
			flagConfigFilePathValue: filepath.Join(wd, "testdata/config-tail1.yaml"),
			flagTailValue:           "2",
			expectedTailValue:       2,
			wantErr:                 false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOptions(genericclioptions.NewTestIOStreamsDiscard())
			fs := pflag.NewFlagSet("", pflag.ExitOnError)
			o.AddFlags(fs)

			args := []string{}
			if tt.flagConfigFilePathValue != "" {
				args = append(args, "--config="+tt.flagConfigFilePathValue)
			}
			if tt.flagTailValue != "" {
				args = append(args, "--tail="+tt.flagTailValue)
			}

			if err := fs.Parse(args); err != nil {
				t.Fatal(err)
			}

			err := o.overrideFlagSetDefaultFromConfig(fs)
			if tt.wantErr {
				if err == nil {
					t.Error("expected err, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}

			if tt.expectedTailValue != o.tail {
				t.Errorf("expected %d for tail, but got %d", tt.expectedTailValue, o.tail)
			}
		})
	}
}

func TestOptionsOverrideFlagSetDefaultFromConfigArray(t *testing.T) {
	tests := []struct {
		config string
		want   []string
	}{
		{
			config: "testdata/config-string.yaml",
			want:   []string{"hello-world"},
		},
		{
			config: "testdata/config-array0.yaml",
			want:   []string{},
		},
		{
			config: "testdata/config-array1.yaml",
			want:   []string{"abcd"},
		},
		{
			config: "testdata/config-array2.yaml",
			want:   []string{"abcd", "efgh"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.config, func(t *testing.T) {
			o := NewOptions(genericclioptions.NewTestIOStreamsDiscard())
			fs := pflag.NewFlagSet("", pflag.ExitOnError)
			o.AddFlags(fs)
			if err := fs.Parse([]string{"--config=" + tt.config}); err != nil {
				t.Fatal(err)
			}
			if err := o.overrideFlagSetDefaultFromConfig(fs); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.want, o.exclude) {
				t.Errorf("expected %v, but got %v", tt.want, o.exclude)
			}
		})
	}

}
