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

package stern

import (
	"bufio"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Tail struct {
	Namespace      string
	PodName        string
	ContainerName  string
	Options        *TailOptions
	closed         chan struct{}
	podColor       *color.Color
	containerColor *color.Color
	tmpl           *template.Template
}

type TailOptions struct {
	Timestamps   bool
	SinceSeconds int64
	Exclude      []*regexp.Regexp
	Include      []*regexp.Regexp
	Namespace    bool
	TailLines    *int64
}

func (o TailOptions) IsExclude(msg string) bool {
	for _, rex := range o.Exclude {
		if rex.MatchString(msg) {
			return true
		}
	}

	return false
}

func (o TailOptions) IsInclude(msg string) bool {
	if len(o.Include) == 0 {
		return true
	}

	for _, rin := range o.Include {
		if rin.MatchString(msg) {
			return true
		}
	}

	return false
}

// NewTail returns a new tail for a Kubernetes container inside a pod
func NewTail(namespace, podName, containerName string, tmpl *template.Template, options *TailOptions) *Tail {
	return &Tail{
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		Options:       options,
		closed:        make(chan struct{}),
		tmpl:          tmpl,
	}
}

var colorList = [][2]*color.Color{
	{color.New(color.FgHiCyan), color.New(color.FgCyan)},
	{color.New(color.FgHiGreen), color.New(color.FgGreen)},
	{color.New(color.FgHiMagenta), color.New(color.FgMagenta)},
	{color.New(color.FgHiYellow), color.New(color.FgYellow)},
	{color.New(color.FgHiBlue), color.New(color.FgBlue)},
	{color.New(color.FgHiRed), color.New(color.FgRed)},
}

func determineColor(podName string) (podColor, containerColor *color.Color) {
	hash := fnv.New32()
	_, _ = hash.Write([]byte(podName))
	idx := hash.Sum32() % uint32(len(colorList))

	colors := colorList[idx]
	return colors[0], colors[1]
}

// Start starts tailing
func (t *Tail) Start(ctx context.Context, i v1.PodInterface) {
	t.podColor, t.containerColor = determineColor(t.PodName)

	go func() {
		g := color.New(color.FgHiGreen, color.Bold).SprintFunc()
		p := t.podColor.SprintFunc()
		c := t.containerColor.SprintFunc()
		if t.Options.Namespace {
			fmt.Fprintf(os.Stderr, "%s %s %s › %s\n", g("+"), p(t.Namespace), p(t.PodName), c(t.ContainerName))
		} else {
			fmt.Fprintf(os.Stderr, "%s %s › %s\n", g("+"), p(t.PodName), c(t.ContainerName))
		}

		req := i.GetLogs(t.PodName, &corev1.PodLogOptions{
			Follow:       true,
			Timestamps:   t.Options.Timestamps,
			Container:    t.ContainerName,
			SinceSeconds: &t.Options.SinceSeconds,
			TailLines:    t.Options.TailLines,
		})

		stream, err := req.Stream(ctx)
		if err != nil {
			fmt.Println(errors.Wrapf(err, "Error opening stream to %s/%s: %s\n", t.Namespace, t.PodName, t.ContainerName))
			return
		}
		defer stream.Close()

		go func() {
			<-t.closed
			stream.Close()
		}()

		if err := t.ConsumeStream(stream, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		close(t.closed)
	}()
}

// Close stops tailing
func (t *Tail) Close() {
	r := color.New(color.FgHiRed, color.Bold).SprintFunc()
	p := t.podColor.SprintFunc()
	if t.Options.Namespace {
		fmt.Fprintf(os.Stderr, "%s %s %s\n", r("-"), p(t.Namespace), p(t.PodName))
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", r("-"), p(t.PodName))
	}
	close(t.closed)
}

// ConsumeStream reads the data from stream reader and writes into the out
// writer.
func (t *Tail) ConsumeStream(stream io.Reader, out io.Writer) error {
	r := bufio.NewReader(stream)
	for {
		line, err := r.ReadBytes('\n')
		msg := string(line)
		// Remove a line break
		msg = strings.TrimSuffix(msg, "\n")

		if !t.Options.IsExclude(msg) && t.Options.IsInclude(msg) {
			t.Print(msg, out)
		}

		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

// Print prints a color coded log message with the pod and container names
func (t *Tail) Print(msg string, out io.Writer) {
	vm := Log{
		Message:        msg,
		Namespace:      t.Namespace,
		PodName:        t.PodName,
		ContainerName:  t.ContainerName,
		PodColor:       t.podColor,
		ContainerColor: t.containerColor,
	}
	if err := t.tmpl.Execute(out, vm); err != nil {
		fmt.Fprintf(os.Stderr, "expanding template failed: %s\n", err)
	}
}

// Log is the object which will be used together with the template to generate
// the output.
type Log struct {
	// Message is the log message itself
	Message string `json:"message"`

	// Namespace of the pod
	Namespace string `json:"namespace"`

	// PodName of the pod
	PodName string `json:"podName"`

	// ContainerName of the container
	ContainerName string `json:"containerName"`

	PodColor       *color.Color `json:"-"`
	ContainerColor *color.Color `json:"-"`
}
