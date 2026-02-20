package parser

import (
	"fmt"
	"regexp"

	"github.com/getpipe-dev/pipe/internal/model"
)

// secretPatterns maps a human-readable description to a regex that matches
// common secrets or credentials accidentally embedded in shell commands.
var secretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AWS access key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"secret assignment", regexp.MustCompile(`(?i)(api_key|secret|token|password)\s*=\s*"?[A-Za-z0-9_/+=\-]{8,}`)},
	{"URL with credentials", regexp.MustCompile(`://[^:]+:[^@]+@`)},
	{"private key header", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"GitHub token", regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`)},
	{"GitLab token", regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`)},
	{"Bearer token", regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-._~+/]+=*`)},
}

// detectSecrets scans all run commands in a step for embedded secrets.
func detectSecrets(s model.Step) []string {
	var findings []string
	check := func(cmd string) {
		for _, sp := range secretPatterns {
			if sp.pattern.MatchString(cmd) {
				findings = append(findings, sp.name)
			}
		}
	}
	if s.Run.IsSingle() {
		check(s.Run.Single)
	}
	for _, cmd := range s.Run.Strings {
		check(cmd)
	}
	for _, sr := range s.Run.SubRuns {
		check(sr.Run)
	}
	return findings
}

// SecretWarnings returns warnings for steps that appear to contain embedded
// secrets but do not have sensitive: true set.
func SecretWarnings(p *model.Pipeline) []string {
	var warns []string
	for _, s := range p.Steps {
		if s.Sensitive {
			continue
		}
		findings := detectSecrets(s)
		if len(findings) > 0 {
			warns = append(warns, fmt.Sprintf(
				"step %q: possible secret detected (%s) — consider adding sensitive: true",
				s.ID, findings[0],
			))
		}
	}
	return warns
}
