package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stern/stern/stern"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestStartPager(t *testing.T) {
	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	o := NewOptions(streams)
	o.pager = "cat"

	config := &stern.Config{Out: o.Out, ErrOut: o.ErrOut}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	closePager, err := o.startPager(cancel, config)
	if err != nil {
		t.Fatal(err)
	}

	if config.Out == o.Out {
		t.Error("expected config.Out to be replaced with a pipe to the pager")
	}

	fmt.Fprintln(config.Out, "log line for the pager")
	closePager()

	if !strings.Contains(out.String(), "log line for the pager") {
		t.Errorf("expected the output to be piped through the pager, but got %q", out.String())
	}
}

func TestStartPagerCancelsContextWhenPagerExits(t *testing.T) {
	streams := genericclioptions.NewTestIOStreamsDiscard()
	o := NewOptions(streams)
	// "true" exits immediately without reading its stdin
	o.pager = "true"

	config := &stern.Config{Out: o.Out, ErrOut: o.ErrOut}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	closePager, err := o.startPager(cancel, config)
	if err != nil {
		t.Fatal(err)
	}
	defer closePager()

	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		t.Error("expected the context to be canceled when the pager exits")
	}
}

func TestStartPagerError(t *testing.T) {
	tests := []struct {
		name  string
		pager string
	}{
		{"unterminated quote", "less '-R"},
		{"empty command", "  "},
		{"command not found", "stern-no-such-pager-command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams := genericclioptions.NewTestIOStreamsDiscard()
			o := NewOptions(streams)
			o.pager = tt.pager

			config := &stern.Config{Out: o.Out, ErrOut: o.ErrOut}

			if _, err := o.startPager(func() {}, config); err == nil {
				t.Error("expected an error, but got none")
			}
		})
	}
}

func TestPagerEnviron(t *testing.T) {
	// t.Setenv registers the restoration of the original values
	// even though the variables are unset later.
	t.Setenv("LESS", "placeholder")
	t.Setenv("LV", "placeholder")

	t.Run("defaults are set when LESS and LV are unset", func(t *testing.T) {
		os.Unsetenv("LESS")
		os.Unsetenv("LV")

		environ := pagerEnviron()
		for _, expected := range []string{"LESS=FRX", "LV=-c"} {
			if !slices.Contains(environ, expected) {
				t.Errorf("expected the environment to contain %q, but got %v", expected, environ)
			}
		}
	})

	t.Run("existing LESS and LV are respected", func(t *testing.T) {
		os.Setenv("LESS", "R")
		os.Setenv("LV", "")

		environ := pagerEnviron()
		for _, unexpected := range []string{"LESS=FRX", "LV=-c"} {
			if slices.Contains(environ, unexpected) {
				t.Errorf("expected the environment not to contain %q, but got %v", unexpected, environ)
			}
		}
		if !slices.Contains(environ, "LESS=R") {
			t.Errorf("expected the environment to contain %q, but got %v", "LESS=R", environ)
		}
	})
}

func TestIsTerminal(t *testing.T) {
	streams := genericclioptions.NewTestIOStreamsDiscard()
	if isTerminal(streams.Out) {
		t.Error("expected false for a non-file writer")
	}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	if isTerminal(devNull) {
		t.Error("expected false for a non-terminal file")
	}
}
