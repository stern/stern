package stern

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
	"text/template"
	"time"
)

func TestConsumeFileTail(t *testing.T) {
	logLines := `line 1
line 2
line 3
line 4`
	tmpl := template.Must(template.New("").Parse(`{{printf "%s\n" .Message}}`))

	tests := []struct {
		name      string
		resumeReq *ResumeRequest
		expected  []byte
	}{
		{
			name: "normal",
			expected: []byte(`line 1
line 2
line 3
line 4
`),
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tail := NewFileTail(tmpl, nil, out, io.Discard, &TailOptions{})
			if err := tail.ConsumeReader(bufio.NewReader(strings.NewReader(logLines))); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected %s, but actual %s", i, tt.expected, out)
			}
		})
	}
}

func TestConsumeFileTailTimestamps(t *testing.T) {
	tmpl := template.Must(template.New("").Parse(`{{if .Timestamp}}{{.Timestamp}} {{end}}{{printf "%s\n" .Message}}`))

	tests := []struct {
		name       string
		line       string
		timestamps bool
		// expected is an exact match. When it is empty the timestamp is
		// generated and cannot be compared, so suffix is checked instead.
		expected string
		suffix   string
	}{
		{
			name:       "a line carrying its own timestamp reuses it",
			line:       "2026-08-20T10:00:00.123456789Z hello\n",
			timestamps: true,
			expected:   "2026-08-20T10:00:00.123456789Z hello\n",
		},
		{
			name:       "a line without one gets the current time",
			line:       "test line\n",
			timestamps: true,
			suffix:     " test line\n",
		},
		{
			name:       "a leading word that is not a timestamp is left in the message",
			line:       "hello world\n",
			timestamps: true,
			suffix:     " hello world\n",
		},
		{
			name:       "without timestamps a timestamped line is untouched",
			line:       "2026-08-20T10:00:00.123456789Z hello\n",
			timestamps: false,
			expected:   "2026-08-20T10:00:00.123456789Z hello\n",
		},
		{
			name:       "without timestamps a plain line is untouched",
			line:       "test line\n",
			timestamps: false,
			expected:   "test line\n",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tail := NewFileTail(tmpl, nil, out, io.Discard, &TailOptions{Timestamps: tt.timestamps, Location: time.UTC})
			if err := tail.ConsumeReader(bufio.NewReader(strings.NewReader(tt.line))); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if tt.expected != "" {
				if out.String() != tt.expected {
					t.Errorf("%d: expected %q, but actual %q", i, tt.expected, out.String())
				}
				return
			}

			if !strings.HasSuffix(out.String(), tt.suffix) {
				t.Errorf("%d: expected output to end with %q, but got %q", i, tt.suffix, out.String())
			}
			generated := strings.TrimSuffix(out.String(), tt.suffix)
			if _, err := time.Parse(TimestampFormatDefault, generated); err != nil {
				t.Errorf("%d: expected a generated timestamp, but got %q: %v", i, generated, err)
			}
		})
	}
}

func TestSplitLogLineIfTimestamped(t *testing.T) {
	tests := []struct {
		line     string
		wantTs   string
		wantRest string
		wantOk   bool
	}{
		{"2026-08-20T10:00:00.123456789Z hello", "2026-08-20T10:00:00.123456789Z", "hello", true},
		{"2026-08-20T10:00:00Z hello", "2026-08-20T10:00:00Z", "hello", true},
		{"2026-08-20T10:00:00+09:00 hello", "2026-08-20T10:00:00+09:00", "hello", true},
		// a plain line must keep its first word, which splitLogLine alone would eat
		{"hello world", "", "hello world", false},
		{"2026-08-20 10:00:00 hello", "", "2026-08-20 10:00:00 hello", false},
		{"nospace", "", "nospace", false},
		{"", "", "", false},
	}

	for i, tt := range tests {
		ts, rest, ok := splitLogLineIfTimestamped(tt.line)
		if ok != tt.wantOk || ts != tt.wantTs || rest != tt.wantRest {
			t.Errorf("%d: splitLogLineIfTimestamped(%q) = (%q, %q, %v), want (%q, %q, %v)",
				i, tt.line, ts, rest, ok, tt.wantTs, tt.wantRest, tt.wantOk)
		}
	}
}
