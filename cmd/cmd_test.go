package cmd

import (
	"strings"
	"testing"

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
			cmd := NewSternCmd(streams)
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err != nil {
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
