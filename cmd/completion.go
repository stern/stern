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
	"os"

	"github.com/spf13/cobra"
)

func runCompletion(shell string, cmd *cobra.Command) error {
	var err error

	switch shell {
	case "bash":
		err = runCompletionBash(cmd)
	case "zsh":
		err = runCompletionZsh(cmd)
	default:
		err = fmt.Errorf("Unsupported shell type: %q", shell)
	}

	return err
}

func runCompletionBash(cmd *cobra.Command) error {
	return cmd.GenBashCompletion(os.Stdout)
}

// runCompletionZsh is based on `kubectl completion zsh`. This function should
// be replaced by cobra implementation when cobra itself supports zsh completion.
// https://github.com/kubernetes/kubernetes/blob/v1.6.1/pkg/kubectl/cmd/completion.go#L136
func runCompletionZsh(cmd *cobra.Command) error {
	out := new(bytes.Buffer)

	zshInitialization := `
__stern_bash_source() {
    alias shopt=':'
    alias _expand=_bash_expand
    alias _complete=_bash_comp
    emulate -L sh
    setopt kshglob noshglob braceexpand
    source "$@"
}
__stern_type() {
    # -t is not supported by zsh
    if [ "$1" == "-t" ]; then
        shift
        # fake Bash 4 to disable "complete -o nospace". Instead
        # "compopt +-o nospace" is used in the code to toggle trailing
        # spaces. We don't support that, but leave trailing spaces on
        # all the time
        if [ "$1" = "__stern_compopt" ]; then
            echo builtin
            return 0
        fi
    fi
    type "$@"
}
__stern_compgen() {
    local completions w
    completions=( $(compgen "$@") ) || return $?
    # filter by given word as prefix
    while [[ "$1" = -* && "$1" != -- ]]; do
        shift
        shift
    done
    if [[ "$1" == -- ]]; then
        shift
    fi
    for w in "${completions[@]}"; do
        if [[ "${w}" = "$1"* ]]; then
            echo "${w}"
        fi
    done
}
__stern_compopt() {
    true # don't do anything. Not supported by bashcompinit in zsh
}
__stern_declare() {
    if [ "$1" == "-F" ]; then
        whence -w "$@"
    else
        builtin declare "$@"
    fi
}
__stern_ltrim_colon_completions()
{
    if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
        # Remove colon-word prefix from COMPREPLY items
        local colon_word=${1%${1##*:}}
        local i=${#COMPREPLY[*]}
        while [[ $((--i)) -ge 0 ]]; do
            COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
        done
    fi
}
__stern_get_comp_words_by_ref() {
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[${COMP_CWORD}-1]}"
    words=("${COMP_WORDS[@]}")
    cword=("${COMP_CWORD[@]}")
}
__stern_filedir() {
    local RET OLD_IFS w qw
    __debug "_filedir $@ cur=$cur"
    if [[ "$1" = \~* ]]; then
        # somehow does not work. Maybe, zsh does not call this at all
        eval echo "$1"
        return 0
    fi
    OLD_IFS="$IFS"
    IFS=$'\n'
    if [ "$1" = "-d" ]; then
        shift
        RET=( $(compgen -d) )
    else
        RET=( $(compgen -f) )
    fi
    IFS="$OLD_IFS"
    IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"
    for w in ${RET[@]}; do
        if [[ ! "${w}" = "${cur}"* ]]; then
            continue
        fi
        if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
            qw="$(__stern_quote "${w}")"
            if [ -d "${w}" ]; then
                COMPREPLY+=("${qw}/")
            else
                COMPREPLY+=("${qw}")
            fi
        fi
    done
}
__stern_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
        printf %q "$1"
    fi
}
autoload -U +X bashcompinit && bashcompinit
# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
    LWORD='\<'
    RWORD='\>'
fi
__stern_convert_bash_to_zsh() {
    sed \
    -e 's/declare -F/whence -w/' \
    -e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
    -e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
    -e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
    -e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
    -e "s/${LWORD}_filedir${RWORD}/__stern_filedir/g" \
    -e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__stern_get_comp_words_by_ref/g" \
    -e "s/${LWORD}__ltrim_colon_completions${RWORD}/__stern_ltrim_colon_completions/g" \
    -e "s/${LWORD}compgen${RWORD}/__stern_compgen/g" \
    -e "s/${LWORD}compopt${RWORD}/__stern_compopt/g" \
    -e "s/${LWORD}declare${RWORD}/__stern_declare/g" \
    -e "s/\\\$(type${RWORD}/\$(__stern_type/g" \
    <<'BASH_COMPLETION_EOF'
`
	out.Write([]byte(zshInitialization))

	if err := cmd.GenBashCompletion(out); err != nil {
		return err
	}

	zshTail := `
BASH_COMPLETION_EOF
}
__stern_bash_source <(__stern_convert_bash_to_zsh)
`
	out.Write([]byte(zshTail))

	fmt.Println(out)

	return nil
}
