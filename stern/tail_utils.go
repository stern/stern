package stern

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Log is the object which will be used together with the template to generate
// the output.
type Log struct {
	// Message is the log message itself
	Message string `json:"message"`

	// Timestamp is the log timestamp, formatted according to --timestamps and
	// --timezone. It is empty when --timestamps is not set. It is kept separate
	// from Message so that templates can still parse Message as JSON.
	Timestamp string `json:"timestamp,omitempty"`

	// Node name of the pod
	NodeName string `json:"nodeName"`

	// Namespace of the pod
	Namespace string `json:"namespace"`

	// PodName of the pod
	PodName string `json:"podName"`

	// ContainerName of the container
	ContainerName string `json:"containerName"`

	// Labels of the pod
	Labels map[string]string `json:"labels"`

	// Annotations of the pod
	Annotations map[string]string `json:"annotations"`

	PodColor       *color.Color `json:"-"`
	ContainerColor *color.Color `json:"-"`
}

type TailOptions struct {
	Timestamps      bool
	TimestampFormat string
	Location        *time.Location

	SinceSeconds *int64
	SinceTime    *metav1.Time
	Exclude      []*regexp.Regexp
	Include      []*regexp.Regexp
	Highlight    []*regexp.Regexp
	Namespace    bool
	TailLines    *int64
	Follow       bool
	OnlyLogLines bool

	// regexp for highlighting the matched string
	reHighlight *regexp.Regexp
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

var colorHighlight = color.New(color.FgRed, color.Bold).SprintFunc()

// reEscape matches the SGR escape sequences that color writes.
var reEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func (o TailOptions) HighlightMatchedString(msg string) string {
	highlight := append(o.Include, o.Highlight...)
	if len(highlight) == 0 {
		return msg
	}

	if o.reHighlight == nil {
		ss := make([]string, len(highlight))
		for i, hl := range highlight {
			ss[i] = "(?:" + hl.String() + ")"
		}

		// We expect a longer match
		sort.Slice(ss, func(i, j int) bool {
			return len(ss[i]) > len(ss[j])
		})

		o.reHighlight = regexp.MustCompile("(" + strings.Join(ss, "|") + ")")
	}

	escapes := reEscape.FindAllStringIndex(msg, -1)
	if len(escapes) == 0 {
		return o.reHighlight.ReplaceAllStringFunc(msg, func(part string) string {
			return colorHighlight(part)
		})
	}

	// Match the text without its escape sequences, so that a match neither
	// cuts a sequence open nor stops at one.
	matches := o.reHighlight.FindAllStringIndex(reEscape.ReplaceAllString(msg, ""), -1)

	// offset converts an offset in that text to an offset in msg.
	offset := func(p int) int {
		for _, e := range escapes {
			if e[0] > p {
				break
			}
			p += e[1] - e[0]
		}
		return p
	}

	var sb strings.Builder
	last := 0
	for _, m := range matches {
		if m[0] == m[1] {
			continue
		}
		start, end := offset(m[0]), offset(m[1]-1)+1
		sb.WriteString(msg[last:start])
		sb.WriteString(colorHighlight(msg[start:end]))
		last = end
	}
	sb.WriteString(msg[last:])

	return sb.String()
}

func (o TailOptions) UpdateTimezoneAndFormat(timestamp string) (string, error) {
	t, err := time.ParseInLocation(time.RFC3339Nano, timestamp, time.UTC)
	if err != nil {
		return "", errors.New("missing timestamp")
	}
	format := TimestampFormatDefault
	if o.TimestampFormat != "" {
		format = o.TimestampFormat
	}
	return t.In(o.Location).Format(format), nil
}
