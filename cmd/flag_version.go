package cmd

import "fmt"

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func buildVersion(version, commit, date string) string {
	result := fmt.Sprintf("version: %s", version)

	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}

	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}

	return result
}
