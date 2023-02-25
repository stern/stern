package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stern/stern/stern"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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
