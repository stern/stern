package cmd

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/kballard/go-shellquote"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/stern/stern/stern"
)

// pagerExtraEnv is appended to the pager's environment unless the variables
// are already set. Similar to git, it makes `less` show raw ANSI colors (R),
// quit if the output fits on one screen (F) and not clear the screen on
// exit (X), and makes `lv` show ANSI colors (-c).
var pagerExtraEnv = map[string]string{
	"LESS": "FRX",
	"LV":   "-c",
}

// startPager starts the pager specified by the --pager flag and replaces
// config.Out with a pipe connected to the pager's stdin. It returns a
// function that closes the pipe and waits for the pager to exit, so that the
// user can keep browsing the output after all logs have been shown. When the
// pager exits, typically when the user quits it, cancel is called to stop
// tailing.
func (o *options) startPager(cancel context.CancelFunc, config *stern.Config) (closePager func(), err error) {
	args, err := shellquote.Split(o.pager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse the pager command")
	}
	if len(args) == 0 {
		return nil, errors.New("the pager command is empty")
	}

	pager := exec.Command(args[0], args[1:]...)
	pager.Env = pagerEnviron()
	pager.Stdout = o.Out
	pager.Stderr = o.ErrOut

	pipe, err := pager.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := pager.Start(); err != nil {
		pipe.Close()
		return nil, errors.Wrap(err, "failed to start the pager")
	}

	config.Out = pipe

	done := make(chan struct{})
	go func() {
		_ = pager.Wait()
		close(done)
		cancel()
	}()

	return func() {
		pipe.Close()
		<-done
	}, nil
}

// pagerEnviron returns the environment for the pager process.
func pagerEnviron() []string {
	environ := os.Environ()
	for name, value := range pagerExtraEnv {
		if _, ok := os.LookupEnv(name); !ok {
			environ = append(environ, name+"="+value)
		}
	}
	return environ
}

// isTerminal returns whether w is a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
