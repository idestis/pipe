package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseDotFile reads a .env file and returns raw key-value pairs.
// Keys are plain names (not PIPE_VAR_ prefixed). Blank lines and lines
// starting with # are skipped. Values may be single- or double-quoted.
// Malformed lines are skipped and reported as warnings.
// Returns os.ErrNotExist naturally when the file is missing.
func ParseDotFile(path string) (map[string]string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close() //nolint:errcheck

	vars := make(map[string]string)
	var warnings []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || line[0] == '#' {
			continue
		}

		// Strip optional "export " prefix.
		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			warnings = append(warnings, fmt.Sprintf("%s:%d: skipping malformed line (missing '='): %q", path, lineNum, line))
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		if !validDotFileKey(key) {
			warnings = append(warnings, fmt.Sprintf("%s:%d: skipping invalid key %q — use only letters, digits, hyphens, and underscores", path, lineNum, key))
			continue
		}

		// Strip matching quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			} else {
				// Unquoted value: strip inline comments.
				value = stripInlineComment(value)
			}
		} else {
			value = stripInlineComment(value)
		}

		vars[key] = value
	}
	if err := scanner.Err(); err != nil {
		return vars, warnings, fmt.Errorf("reading %s: %w", path, err)
	}
	return vars, warnings, nil
}

// stripInlineComment removes a trailing # comment from an unquoted value.
func stripInlineComment(s string) string {
	if idx := strings.IndexByte(s, '#'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// validDotFileKey checks that a key contains only letters, digits, hyphens,
// and underscores, and is non-empty.
func validDotFileKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}
