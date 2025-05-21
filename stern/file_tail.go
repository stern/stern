package stern

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/fatih/color"
)

type FileTail struct {
	Options *TailOptions
	tmpl    *template.Template
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
}

// NewFileTail returns a new tail of the input reader
func NewFileTail(tmpl *template.Template, in io.Reader, out, errOut io.Writer, options *TailOptions) *FileTail {
	return &FileTail{
		Options: options,
		tmpl:    tmpl,
		in:      in,
		out:     out,
		errOut:  errOut,
	}
}

// Start starts tailing
func (t *FileTail) Start() error {
	reader := bufio.NewReader(t.in)
	err := t.ConsumeReader(reader)

	return err
}

// ConsumeReader reads the data from the reader and writes into the out
// writer.
func (t *FileTail) ConsumeReader(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) != 0 {
			t.consumeLine(strings.TrimSuffix(string(line), "\n"))
		}

		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

func (t *FileTail) sprint(msg string) (string, error) {
	vm := Log{
		Message:        msg,
		NodeName:       "",
		Namespace:      "",
		PodName:        "",
		ContainerName:  "",
		PodColor:       color.New(color.Reset),
		ContainerColor: color.New(color.Reset),
	}

	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, vm); err != nil {
		return "", fmt.Errorf("expanding template failed: %s", err)
	}

	return buf.String(), nil
}

// Print prints a color coded log message
func (t *FileTail) Print(msg string) {
	buf, err := t.sprint(msg)
	if err != nil {
		fmt.Fprintf(t.errOut, "%s\n", err)
		return
	}

	fmt.Fprint(t.out, t.Options.HighlightMatchedString(buf))
}

// PrintWithoutHighlight prints a log message without applying any highlight.
func (t *FileTail) PrintWithoutHighlight(msg string) {
	buf, err := t.sprint(msg)
	if err != nil {
		fmt.Fprintf(t.errOut, "%s\n", err)
		return
	}

	fmt.Fprint(t.out, buf)
}

func (t *FileTail) consumeLine(line string) {
	content := line

	if t.Options.IsExclude(content) || !t.Options.IsInclude(content) {
		return
	}

	t.Print(content)
}
