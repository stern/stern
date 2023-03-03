package cmd

import (
	"bytes"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/stern/stern/stern"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
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
			"One of pod-query, --selector, --field-selector or --prompt is required",
		},
		{
			"Specify both selector and resource",
			func() *options {
				o := NewOptions(streams)
				o.selector = "app=nginx"
				o.resource = "deployment/nginx"

				return o
			}(),
			"--selector and the <resource>/<name> query can not be set at the same time",
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
	fieldSelector, _ := fields.ParseSelector("f=field")

	re := regexp.MustCompile

	defaultConfig := func() *stern.Config {
		return &stern.Config{
			KubeConfig:            "",
			ContextName:           "",
			Namespaces:            []string{},
			PodQuery:              re(""),
			ExcludePodQuery:       nil,
			Timestamps:            false,
			Location:              local,
			ContainerQuery:        re(".*"),
			ExcludeContainerQuery: nil,
			ContainerStates:       []stern.ContainerState{stern.ALL_STATES},
			Exclude:               nil,
			Include:               nil,
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
				o.kubeConfig = "kubeconfig1"
				o.context = "context1"
				o.namespaces = []string{"ns1", "ns2"}
				o.podQuery = "query1"
				o.excludePod = []string{"exp1", "exp2"}
				o.timestamps = true
				o.timezone = "UTC" // Location
				o.container = "container1"
				o.excludeContainer = []string{"exc1", "exc2"}
				o.containerStates = []string{"running", "terminated"}
				o.exclude = []string{"ex1", "ex2"}
				o.include = []string{"in1", "in2"}
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

				return o
			}(),
			func() *stern.Config {
				c := defaultConfig()
				c.KubeConfig = "kubeconfig1"
				c.ContextName = "context1"
				c.Namespaces = []string{"ns1", "ns2"}
				c.PodQuery = re("query1")
				c.ExcludePodQuery = []*regexp.Regexp{re("exp1"), re("exp2")}
				c.Timestamps = true
				c.Location = utc
				c.ContainerQuery = re("container1")
				c.ExcludeContainerQuery = []*regexp.Regexp{re("exc1"), re("exc2")}
				c.ContainerStates = []stern.ContainerState{stern.RUNNING, stern.TERMINATED}
				c.Exclude = []*regexp.Regexp{re("ex1"), re("ex2")}
				c.Include = []*regexp.Regexp{re("in1"), re("in2")}
				c.InitContainers = false
				c.EphemeralContainers = false
				c.Since = 1 * time.Hour
				c.AllNamespaces = true
				c.LabelSelector = labelSelector
				c.FieldSelector = fieldSelector
				c.TailLines = pointer.Int64(10)
				c.Follow = false
				c.Resource = "res1"
				c.OnlyLogLines = true
				c.MaxLogRequests = 30

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
