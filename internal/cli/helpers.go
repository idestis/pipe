package cli

import (
	"errors"
	"os"
	"strings"
)

func validName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			// always allowed
		case c == '-' || c == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// friendlyError converts common OS errors into user-friendly messages.
func friendlyError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "permission denied — check directory permissions for ~/.pipe"
	}
	return err.Error()
}

// isYAMLError returns true if the error originated from YAML parsing.
func isYAMLError(err error) bool {
	return strings.Contains(err.Error(), "parsing pipeline")
}

// unwrapYAMLError extracts the YAML-specific error detail from a wrapped
// "parsing pipeline" error, stripping the redundant prefix.
func unwrapYAMLError(err error) error {
	msg := err.Error()
	// Strip our wrapping prefix "parsing pipeline \"name\": " to get the yaml detail.
	if i := strings.Index(msg, "parsing pipeline"); i >= 0 {
		// Find the ": " after the closing quote of the pipeline name.
		rest := msg[i:]
		if j := strings.Index(rest, ": "); j >= 0 {
			detail := rest[j+2:]
			return errors.New(detail)
		}
	}
	return err
}
