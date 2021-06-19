//   Copyright 2017 Wercker Holding BV
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

package cmd

import (
	"bytes"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func runCompletion(shell string, cmd *cobra.Command, out io.Writer) error {
	var err error

	switch shell {
	case "bash":
		err = cmd.GenBashCompletion(out)
	case "zsh":
		err = runCompletionZsh(cmd, out)
	default:
		err = fmt.Errorf("Unsupported shell type: %q", shell)
	}

	return err
}

// runCompletionZsh is based on `kubectl completion zsh`. This function should
// be replaced by cobra implementation when cobra itself supports zsh completion.
// https://github.com/kubernetes/kubernetes/blob/v1.6.1/pkg/kubectl/cmd/completion.go#L136
func runCompletionZsh(cmd *cobra.Command, out io.Writer) error {
	b := new(bytes.Buffer)
	if err := cmd.GenZshCompletion(b); err != nil {
		return err
	}

	// Cobra doesn't source zsh completion file, explicitly doing it here
	fmt.Fprintf(b, "compdef _stern stern")

	fmt.Fprint(out, b.String())

	return nil
}
