package stern

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
	"text/template"
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
			tail := NewFileTail(tmpl, "test-file", nil, out, io.Discard, &TailOptions{})
			if err := tail.ConsumeReader(bufio.NewReader(strings.NewReader(logLines))); err != nil {
				t.Fatalf("%d: unexpected err %v", i, err)
			}

			if !bytes.Equal(tt.expected, out.Bytes()) {
				t.Errorf("%d: expected %s, but actual %s", i, tt.expected, out)
			}
		})
	}
}

func TestFilePrintStarting(t *testing.T) {
	tests := []struct {
		options  *TailOptions
		expected []byte
	}{
		{
			&TailOptions{},
			[]byte("+ › test-file\n"),
		},
		{
			&TailOptions{
				OnlyLogLines: true,
			},
			[]byte{},
		},
	}

	for i, tt := range tests {
		errOut := new(bytes.Buffer)
		tail := NewFileTail(nil, "test-file", strings.NewReader(""), io.Discard, errOut, tt.options)
		tail.printStarting()

		if !bytes.Equal(tt.expected, errOut.Bytes()) {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, errOut)
		}
	}
}

func TestFilePrintStopping(t *testing.T) {
	tests := []struct {
		options  *TailOptions
		expected []byte
	}{
		{
			&TailOptions{},
			[]byte("- › test-file\n"),
		},
		{
			&TailOptions{
				OnlyLogLines: true,
			},
			[]byte{},
		},
	}

	for i, tt := range tests {
		errOut := new(bytes.Buffer)
		tail := NewFileTail(nil, "test-file", strings.NewReader(""), io.Discard, errOut, tt.options)
		tail.printStopping()

		if !bytes.Equal(tt.expected, errOut.Bytes()) {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, errOut)
		}
	}
}
