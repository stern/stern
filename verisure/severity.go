package verisure

import (
	"errors"
	"strings"
)

type Severity string

const (
	ALL_LEVELS = ""
	INFO       = "Informational"
	WARNING    = "Warning"
	DEBUG      = "Debug"
	ERROR      = "Error"
)

func NewSeverity(severityName string) (Severity, error) {
	switch strings.ToLower(severityName) {
	case ALL_LEVELS:
		return ALL_LEVELS, nil
	case "informational", "info":
		return INFO, nil
	case "debug":
		return DEBUG, nil
	case "error":
		return ERROR, nil
	case "warning":
		return WARNING, nil
	default:
		return "", errors.New("Unknown severity level")
	}
}

func (wantedSeverity Severity) Match(actualSeverity Severity) bool {
	if wantedSeverity == ALL_LEVELS || wantedSeverity == actualSeverity {
		return true
	}
	return false
}
