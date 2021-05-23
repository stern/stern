package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/stern/stern/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	targetSectionBegin = "<!-- auto generated cli flags begin --->"
	targetSectionEnd   = "<!-- auto generated cli flags end --->"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "readmePath argument required")
		os.Exit(1)
	}
	readmePath := os.Args[1]

	flagsMarkdownTable := GenerateFlagsMarkdownTable()

	readmeString, err := GenerateReadme(readmePath, flagsMarkdownTable)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = OverwriteReadme(readmePath, readmeString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GenerateFlagsMarkdownTable generates markdown table of flag list.
// This function loads flag list and generates such as following text:
//  flag            | default         | purpose
// -----------------|-----------------|---------
//  `--flag1`, `-f` |                 | This is flag1.
//  `--flag2`       | `flag2-default` | This is flag2.
func GenerateFlagsMarkdownTable() string {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	o := cmd.NewOptions(genericclioptions.NewTestIOStreamsDiscard())
	o.AddFlags(fs)

	flagMaxlen, defaultMaxlen := len(" flag "), len(" default ")
	allTexts := make([][]string, 0)
	fs.VisitAll(func(flag *pflag.Flag) {
		// won't append deprecated flag
		if flag.Deprecated != "" {
			return
		}

		flagText := ""
		if flag.Shorthand != "" {
			flagText = fmt.Sprintf(" `--%s`, `-%s` ", flag.Name, flag.Shorthand)
		} else {
			flagText = fmt.Sprintf(" `--%s` ", flag.Name)
		}
		if len(flagText) > flagMaxlen {
			flagMaxlen = len(flagText)
		}

		flagTypeName, usage := pflag.UnquoteUsage(flag)
		defaultText := ""
		if flag.DefValue != "" {
			switch flagTypeName {
			// convert []string{"aaa", "bbb"} to "aaa,bbb"
			case "strings":
				stirngSlice, err := fs.GetStringSlice(flag.Name)
				if err != nil {
					panic(err)
				}

				defaultValuesString := ""
				for _, s := range stirngSlice {
					defaultValuesString += fmt.Sprintf("%s,", s)
				}
				defaultValuesString = strings.TrimRight(defaultValuesString, ",")

				if defaultValuesString != "" {
					defaultText += fmt.Sprintf(" `%s` ", defaultValuesString)
				}
			default:
				defaultText += fmt.Sprintf(" `%s` ", flag.DefValue)
			}
		}
		if len(defaultText) > defaultMaxlen {
			defaultMaxlen = len(defaultText)
		}

		purposeText := fmt.Sprintf(" %s", usage)

		allTexts = append(allTexts, []string{flagText, defaultText, purposeText})
	})

	tableText := fmt.Sprintf(
		" flag %s| default %s| purpose\n",
		strings.Repeat(" ", flagMaxlen-len(" flag ")),
		strings.Repeat(" ", defaultMaxlen-len(" default ")))
	tableText += fmt.Sprintf(
		"%s|%s|%s\n",
		strings.Repeat("-", flagMaxlen),
		strings.Repeat("-", defaultMaxlen),
		strings.Repeat("-", len(" purpose ")))
	for _, text := range allTexts {
		tableText += text[0]
		tableText += strings.Repeat(" ", flagMaxlen-len(text[0]))
		tableText += "|" + text[1]
		tableText += strings.Repeat(" ", defaultMaxlen-len(text[1]))
		tableText += "|" + text[2]
		tableText += "\n"
	}

	return tableText
}

// GenerateReadme generates README.md in which flags markdown table is embedded.
// Overwrite the section which specified as the target.
func GenerateReadme(readmePath, flagsMarkdownTable string) (string, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	inTargetSection := false
	tableText := ""
	scanner := bufio.NewScanner(strings.NewReader(string(buf)))
	for scanner.Scan() {
		if !inTargetSection {
			tableText += scanner.Text() + "\n"
		}

		if scanner.Text() == targetSectionBegin {
			inTargetSection = true
		}

		if scanner.Text() == targetSectionEnd {
			tableText += flagsMarkdownTable
			tableText += scanner.Text() + "\n"
			inTargetSection = false
		}
	}

	return tableText, nil
}

// OverwriteReadme overwrites README.md with passed readmeString.
func OverwriteReadme(readmePath, readmeString string) error {
	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(readmeString)
	if err != nil {
		return err
	}

	return nil
}
