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
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type Tail struct {
	clientset corev1client.CoreV1Interface

	NodeName       string
	Namespace      string
	PodName        string
	ContainerName  string
	Options        *TailOptions
	closed         chan struct{}
	podColor       *color.Color
	containerColor *color.Color
	timestampColor *color.Color
	callerColor    *color.Color
	tmpl           *template.Template
	out            io.Writer
	errOut         io.Writer
}

type TailOptions struct {
	Timestamps bool
	Location   *time.Location

	SinceSeconds int64
	Exclude      []*regexp.Regexp
	Include      []*regexp.Regexp
	Namespace    bool
	TailLines    *int64
	Follow       bool
	OnlyLogLines bool
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

func (o TailOptions) UpdateTimezoneIfNeeded(message string) (string, error) {
	if !o.Timestamps {
		return message, nil
	}

	idx := strings.IndexRune(message, ' ')
	if idx == -1 {
		return message, errors.New("missing timestamp")
	}

	datetime := message[:idx]
	t, err := time.ParseInLocation(time.RFC3339Nano, datetime, time.UTC)
	if err != nil {
		return message, errors.New("missing timestamp")
	}

	return t.In(o.Location).Format("2006-01-02T15:04:05.000000000Z07:00") + message[idx:], nil
}

// NewTail returns a new tail for a Kubernetes container inside a pod
func NewTail(clientset corev1client.CoreV1Interface, nodeName, namespace, podName, containerName string, tmpl *template.Template, out, errOut io.Writer, options *TailOptions) *Tail {
	podColor, containerColor := determineColor(podName)

	return &Tail{
		clientset:      clientset,
		NodeName:       nodeName,
		Namespace:      namespace,
		PodName:        podName,
		ContainerName:  containerName,
		Options:        options,
		closed:         make(chan struct{}),
		tmpl:           tmpl,
		podColor:       podColor,
		containerColor: containerColor,
		timestampColor: color.New(color.FgGreen),
		callerColor:    color.New(color.FgCyan),

		out:    out,
		errOut: errOut,
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
func (t *Tail) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-t.closed
		cancel()
	}()

	t.printStarting()

	req := t.clientset.Pods(t.Namespace).GetLogs(t.PodName, &corev1.PodLogOptions{
		Follow:       t.Options.Follow,
		Timestamps:   t.Options.Timestamps,
		Container:    t.ContainerName,
		SinceSeconds: &t.Options.SinceSeconds,
		TailLines:    t.Options.TailLines,
	})

	err := t.ConsumeRequest(ctx, req)

	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

// Close stops tailing
func (t *Tail) Close() {
	t.printStopping()

	close(t.closed)
}

func (t *Tail) printStarting() {
	if !t.Options.OnlyLogLines {
		g := color.New(color.FgHiGreen, color.Bold).SprintFunc()
		p := t.podColor.SprintFunc()
		c := t.containerColor.SprintFunc()
		if t.Options.Namespace {
			fmt.Fprintf(t.errOut, "%s %s %s › %s\n", g("+"), p(t.Namespace), p(t.PodName), c(t.ContainerName))
		} else {
			fmt.Fprintf(t.errOut, "%s %s › %s\n", g("+"), p(t.PodName), c(t.ContainerName))
		}
	}
}

func (t *Tail) printStopping() {
	if !t.Options.OnlyLogLines {
		r := color.New(color.FgHiRed, color.Bold).SprintFunc()
		p := t.podColor.SprintFunc()
		c := t.containerColor.SprintFunc()
		if t.Options.Namespace {
			fmt.Fprintf(t.errOut, "%s %s %s › %s\n", r("-"), p(t.Namespace), p(t.PodName), c(t.ContainerName))
		} else {
			fmt.Fprintf(t.errOut, "%s %s › %s\n", r("-"), p(t.PodName), c(t.ContainerName))
		}
	}
}

// ConsumeRequest reads the data from request and writes into the out
// writer.
func (t *Tail) ConsumeRequest(ctx context.Context, request rest.ResponseWrapper) error {
	stream, err := request.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	r := bufio.NewReader(stream)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) != 0 {
			msg := string(line)
			// Remove a line break
			msg = strings.TrimSuffix(msg, "\n")

			if t.Options.IsExclude(msg) || !t.Options.IsInclude(msg) {
				continue
			}

			msg, err := t.Options.UpdateTimezoneIfNeeded(msg)
			if err != nil {
				t.Print(fmt.Sprintf("[%v] %s", err, msg))
				continue
			}

			t.Print(msg)
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
func (t *Tail) Print(msg string) {
	vm := Log{
		Message:        msg,
		NodeName:       t.NodeName,
		Namespace:      t.Namespace,
		PodName:        t.PodName,
		ContainerName:  t.ContainerName,
		PodColor:       t.podColor,
		ContainerColor: t.containerColor,
		TimestampColor: t.timestampColor,
		CallerColor:    t.callerColor,
	}

	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, vm); err != nil {
		fmt.Fprintf(t.errOut, "expanding template failed: %s\n", err)
		return
	}

	fmt.Fprint(t.out, buf.String())
}

// Log is the object which will be used together with the template to generate
// the output.
type Log struct {
	// Message is the log message itself
	Message string `json:"message"`

	// Node name of the pod
	NodeName string `json:"nodeName"`

	// Namespace of the pod
	Namespace string `json:"namespace"`

	// PodName of the pod
	PodName string `json:"podName"`

	// ContainerName of the container
	ContainerName string `json:"containerName"`

	PodColor       *color.Color `json:"-"`
	ContainerColor *color.Color `json:"-"`
	TimestampColor *color.Color `json:"-"`
	CallerColor    *color.Color `json:"-"`
}
