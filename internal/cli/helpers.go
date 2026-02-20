package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/auth"
	"github.com/getpipe-dev/pipe/internal/hub"
	"github.com/spf13/cobra"
)

var (
	tagRegex             = regexp.MustCompile(`^[a-z0-9]([a-z0-9.\-]*[a-z0-9])?$`)
	consecutiveSpecialRe = regexp.MustCompile(`[.\-]{2}`)
)

func validTag(tag string) error {
	if len(tag) == 0 || len(tag) > 128 {
		return fmt.Errorf("must be 1-128 characters")
	}
	if !tagRegex.MatchString(tag) {
		return fmt.Errorf("must start and end with a lowercase letter or digit, and contain only lowercase letters, digits, hyphens, and dots")
	}
	if consecutiveSpecialRe.MatchString(tag) {
		return fmt.Errorf("must not contain consecutive hyphens or dots")
	}
	return nil
}

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

// validOwner checks that an owner/username contains only lowercase letters,
// digits, hyphens, and dots, cannot start with a hyphen or dot, and must be
// between 4 and 30 characters long. This matches the hub's username rules.
func validOwner(name string) bool {
	if len(name) < 4 || len(name) > 30 {
		return false
	}
	for i, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			// always allowed
		case c == '-' || c == '.':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// validVarKey checks that a variable key contains only letters, digits, hyphens,
// and underscores, and is non-empty.
func validVarKey(key string) bool {
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

// parseVarOverrides parses KEY=value pairs from CLI args into a map.
func parseVarOverrides(args []string) (map[string]string, error) {
	overrides := make(map[string]string)
	for _, arg := range args {
		idx := strings.IndexByte(arg, '=')
		if idx < 0 {
			return nil, fmt.Errorf("invalid variable override %q — expected KEY=value", arg)
		}
		key := arg[:idx]
		value := arg[idx+1:]
		if !validVarKey(key) {
			return nil, fmt.Errorf("invalid variable key %q — use only letters, digits, hyphens, and underscores", key)
		}
		overrides[key] = value
	}
	return overrides, nil
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

// short safely truncates s to at most n characters for display.
func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func exactArgs(n int, usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("usage: %s", usage)
		}
		return nil
	}
}

func rangeArgs(min, max int, usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < min || len(args) > max {
			return fmt.Errorf("usage: %s", usage)
		}
		return nil
	}
}

func noArgs(cmd string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown arguments — usage: %s", cmd)
		}
		return nil
	}
}

func maxArgs(max int, usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > max {
			return fmt.Errorf("too many arguments — usage: %s", usage)
		}
		return nil
	}
}

// requireAuth loads credentials and returns them, or an error if not logged in.
func requireAuth() (*auth.Credentials, error) {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}
	if creds == nil {
		return nil, fmt.Errorf("not logged in — run \"pipe login\" first")
	}
	return creds, nil
}

// newDefaultHubClient creates an unauthenticated hub API client.
func newDefaultHubClient() *hub.Client {
	log.Debug("creating unauthenticated hub client", "baseURL", apiURL)
	return hub.NewClient(apiURL, "")
}

// newHubClient creates a hub API client from stored credentials.
func newHubClient(creds *auth.Credentials) *hub.Client {
	baseURL := creds.APIBaseURL
	if baseURL == "" {
		baseURL = apiURL
	}
	log.Debug("creating authenticated hub client", "baseURL", baseURL)
	return hub.NewClient(baseURL, creds.APIKey)
}
