package cmd

import (
	"fmt"
	"io"
)

var (
	version = "verisure-dev"
	commit  = ""
	date    = ""
)

func outputVersionInfo(out io.Writer) {
	fmt.Fprintf(out, "version: %s\n", version)

	if commit != "" {
		fmt.Fprintf(out, "commit: %s\n", commit)
	}

	if date != "" {
		fmt.Fprintf(out, "built at: %s\n", date)
	}
}
