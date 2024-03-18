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
	inName  string
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
}

func NewFileTail(tmpl *template.Template, inName string, in io.Reader, out, errOut io.Writer, options *TailOptions) *FileTail {
	return &FileTail{
		Options: options,
		tmpl:    tmpl,
		inName:  inName,
		in:      in,
		out:     out,
		errOut:  errOut,
	}
}

var fileColor *color.Color = color.New(color.FgYellow)

// Start starts tailing
func (t *FileTail) Start() error {
	t.printStarting()

	reader := bufio.NewReader(t.in)
	err := t.ConsumeReader(reader)

	return err
}

// Close stops tailing
func (t *FileTail) Close() {
	t.printStopping()
}

func (t *FileTail) printStarting() {
	if !t.Options.OnlyLogLines {
		g := color.New(color.FgHiGreen, color.Bold).SprintFunc()
		y := fileColor.SprintFunc()
		fmt.Fprintf(t.errOut, "%s › %s\n", g("+"), y(t.inName))
	}
}

func (t *FileTail) printStopping() {
	if !t.Options.OnlyLogLines {
		r := color.New(color.FgHiRed, color.Bold).SprintFunc()
		y := fileColor.SprintFunc()
		fmt.Fprintf(t.errOut, "%s › %s\n", r("-"), y(t.inName))
	}
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

// Print prints a color coded log message
func (t *FileTail) Print(msg string) {
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
		fmt.Fprintf(t.errOut, "expanding template failed: %s\n", err)
		return
	}

	fmt.Fprint(t.out, buf.String())
}

func (t *FileTail) consumeLine(line string) {
	content := line

	if t.Options.IsExclude(content) || !t.Options.IsInclude(content) {
		return
	}

	msg := t.Options.HighlightMatchedString(content)
	t.Print(msg)
}
