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
	"encoding/json"
	"errors"
	"fmt"
  //"runtime/debug"
	"github.com/fatih/color"
	"github.com/henriknelson/verisure"
	"hash/fnv"
	"io"
	"strings"
	"text/template"
	"time"
	"unicode"
	//  "github.com/gookit/goutil/dump"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// RFC3339Nano with trailing zeros
const TimestampFormatDefault = "2006-01-02T15:04:05.000000000Z07:00"

// time.DateTime without year
const TimestampFormatShort = "01-02 15:04:05"

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
	tmpl           *template.Template
	last           struct {
		timestamp string // RFC3339 timestamp (not RFC3339Nano)
		lines     int    // the number of lines seen during this timestamp
	}
	resumeRequest *ResumeRequest
	out           io.Writer
	errOut        io.Writer
}

type ResumeRequest struct {
	Timestamp   string // RFC3339 timestamp (not RFC3339Nano)
	LinesToSkip int    // the number of lines to skip during this timestamp
}

// NewTail returns a new tail for a Kubernetes container inside a pod
func NewTail(clientset corev1client.CoreV1Interface, nodeName, namespace, podName, containerName string, tmpl *template.Template, out, errOut io.Writer, options *TailOptions, diffContainer bool) *Tail {
	podColor, containerColor := determineColor(podName, containerName, diffContainer)

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

		out:    out,
		errOut: errOut,
	}
}

func determineColor(podName, containerName string, diffContainer bool) (podColor, containerColor *color.Color) {
	colors := colorList[colorIndex(podName)]
	if diffContainer {
		return colors[0], colorList[colorIndex(containerName)][1]
	}
	return colors[0], colors[1]
}

func colorIndex(name string) uint32 {
	hash := fnv.New32()
	_, _ = hash.Write([]byte(name))
	return hash.Sum32() % uint32(len(colorList))
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
		Timestamps:   true,
		Container:    t.ContainerName,
		SinceSeconds: t.Options.SinceSeconds,
		SinceTime:    t.Options.SinceTime,
		TailLines:    t.Options.TailLines,
	})

	err := t.ConsumeRequest(ctx, req)

	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

func (t *Tail) Resume(ctx context.Context, resumeRequest *ResumeRequest) error {
	sinceTime, err := resumeRequest.sinceTime()
	if err != nil {
		fmt.Fprintf(t.errOut, "failed to resume: %s, fallback to Start()\n", err)
		return t.Start(ctx)
	}
	t.resumeRequest = resumeRequest
	t.Options.SinceTime = sinceTime
	t.Options.SinceSeconds = nil
	t.Options.TailLines = nil
	return t.Start(ctx)
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
  //debug.PrintStack()
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
			if t.Options.Output == "verisure" {
				t.consumeVerisureLine(strings.TrimSuffix(string(line), "\n"))
			} else {
				t.consumeLine(strings.TrimSuffix(string(line), "\n"))
			}
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
	}

	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, vm); err != nil {
		fmt.Fprintf(t.errOut, "expanding template failed: %s\n", err)
		return
	}

	fmt.Fprint(t.out, buf.String())

}

func (t *Tail) GetResumeRequest() *ResumeRequest {
	if t.last.timestamp == "" {
		return nil
	}
	return &ResumeRequest{Timestamp: t.last.timestamp, LinesToSkip: t.last.lines}
}

func (t *Tail) consumeLine(line string) {

	rfc3339Nano, content, err := splitLogLine(line)
	if err != nil {
		t.Print(fmt.Sprintf("[%v] %s", err, line))
		return
	}

	// PodLogOptions.SinceTime is RFC3339, not RFC3339Nano.
	// We convert it to RFC3339 to skip the lines seen during this timestamp when resuming.
	rfc3339 := removeSubsecond(rfc3339Nano)
	t.rememberLastTimestamp(rfc3339)
	if t.resumeRequest.shouldSkip(rfc3339) {
		return
	}

	if t.Options.IsExclude(content) || !t.Options.IsInclude(content) {
		return
	}

	msg := t.Options.HighlightMatchedString(content)

	if t.Options.Timestamps {
		updatedTs, err := t.Options.UpdateTimezoneAndFormat(rfc3339Nano)
		if err != nil {
			t.Print(fmt.Sprintf("[%v] %s", err, line))
			return
		}
		msg = updatedTs + " " + msg
	}

	t.Print(msg)
}

func (t *Tail) consumeVerisureLine(line string) {

	rfc3339Nano, content, err := splitLogLine(line)
	if err != nil {
		t.Print(fmt.Sprintf("[%v] %s", err, line))
		return
	}

	// PodLogOptions.SinceTime is RFC3339, not RFC3339Nano.
	// We convert it to RFC3339 to skip the lines seen during this timestamp when resuming.
	rfc3339 := removeSubsecond(rfc3339Nano)
	t.rememberLastTimestamp(rfc3339)
	if t.resumeRequest.shouldSkip(rfc3339) {
		return
	}

  if ! json.Valid([]byte(content)) {
    t.Print("")
    fmt.Println(content)
    return
  }

    var logItem verisure.LogItem
  	err = json.Unmarshal([]byte(content), &logItem)
    if err != nil {
  		t.Print(fmt.Sprintf("[%v] %s", err, line))
  		return
  	}


  	if t.Options.IsExclude(logItem.Message) || !t.Options.IsInclude(logItem.Message) {
  		return
  	}

  	msg := t.Options.HighlightMatchedString(logItem.Message)

  	var filtered bool
  	filtered = false

  	if match, newMessage := t.Options.ContainsFilter(logItem.Message); match {
      logItem.Message = newMessage
  		filtered = true
  	}

  	if logItem.Source != nil {
  		for key, value := range logItem.Source {
  			if match, newValue := t.Options.ContainsFilter(fmt.Sprintf("%v", value)); match {
  				logItem.Source[key] = newValue
  				filtered = true
  			}
  		}
  	}

  	if logItem.Extended != nil {
  		for key, value := range logItem.Extended {
  			if match, newValue := t.Options.ContainsFilter(fmt.Sprintf("%v", value)); match {
  				logItem.Extended[key] = newValue
  				filtered = true
  			}
  		}
  	}

  	if ! filtered {
  		return
  	}

  	if t.Options.Timestamps {
  		updatedTs, err := t.Options.UpdateTimezoneAndFormat(rfc3339Nano)
  		if err != nil {
  			t.Print(fmt.Sprintf("[%v] %s", err, line))
  			return
  		}
  		msg = updatedTs + " " + msg
  	}

  	wantedLevel, err := verisure.NewSeverity(t.Options.Level)
  	if err != nil {
  		return
  	}

  	if ! wantedLevel.Match(logItem.Level) {
  		return
  	}

    //fmt.Printf("%s", line)
  	t.Print(msg)
  	logItem.Print(t.out)
}

func (t *Tail) rememberLastTimestamp(timestamp string) {
	if t.last.timestamp == timestamp {
		t.last.lines++
		return
	}
	t.last.timestamp = timestamp
	t.last.lines = 1
}

func (r *ResumeRequest) sinceTime() (*metav1.Time, error) {
	sinceTime, err := time.Parse(time.RFC3339, r.Timestamp)

	if err != nil {
		return nil, err
	}
	metaTime := metav1.NewTime(sinceTime)
	return &metaTime, nil
}

func (r *ResumeRequest) shouldSkip(timestamp string) bool {
	if r == nil {
		return false
	}
	if r.Timestamp == "" {
		return false
	}
	if r.Timestamp != timestamp {
		return false
	}
	if r.LinesToSkip <= 0 {
		return false
	}
	r.LinesToSkip--
	return true
}

func splitLogLine(line string) (timestamp string, content string, err error) {
	idx := strings.IndexRune(line, ' ')
	if idx == -1 {
		return "", "", errors.New("missing timestamp")
	}
	return line[:idx], line[idx+1:], nil
}

// removeSubsecond removes the subsecond of the timestamp.
// It converts RFC3339Nano to RFC3339 fast.
func removeSubsecond(timestamp string) string {
	dot := strings.IndexRune(timestamp, '.')
	if dot == -1 {
		return timestamp
	}
	var last int
	for i := dot; i < len(timestamp); i++ {
		if unicode.IsDigit(rune(timestamp[i])) {
			last = i
		}
	}
	if last == 0 {
		return timestamp
	}
	return timestamp[:dot] + timestamp[last+1:]
}
