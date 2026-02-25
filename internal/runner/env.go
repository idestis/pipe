package runner

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// EnvKey builds a PIPE_* environment variable name from step/sub-run IDs.
// Hyphens become underscores, everything uppercased.
func EnvKey(parts ...string) string {
	joined := strings.Join(parts, "_")
	joined = strings.ReplaceAll(joined, "-", "_")
	return "PIPE_" + strings.ToUpper(joined)
}

// VarEnvKey builds a PIPE_VAR_* environment variable name from a user-defined
// variable key. Hyphens become underscores, everything uppercased.
func VarEnvKey(key string) string {
	k := strings.ReplaceAll(key, "-", "_")
	return "PIPE_VAR_" + strings.ToUpper(k)
}

// sysEnvMap returns the current system environment as a flat map.
func sysEnvMap() map[string]string {
	m := make(map[string]string)
	for _, entry := range os.Environ() {
		if k, v, ok := strings.Cut(entry, "="); ok {
			m[k] = v
		}
	}
	return m
}

// defaultFunc is a Go template helper: {{ .VAR | default "fallback" }}.
// Go template pipes pass the piped value as the last argument, so the
// signature is defaultFunc(fallback, val).
func defaultFunc(fallback string, val any) string {
	if val == nil {
		return fallback
	}
	s := fmt.Sprint(val)
	if s != "" {
		return s
	}
	return fallback
}

// renderVarValue treats value as a Go text/template, executing it with the
// system environment as the data context. On any parse/exec error the
// original value is returned unchanged (graceful degradation).
func renderVarValue(value string, sysEnv map[string]string) string {
	// Fast path: no template delimiters at all.
	if !strings.Contains(value, "{{") {
		return value
	}
	tmpl, err := template.New("var").
		Funcs(template.FuncMap{"default": defaultFunc}).
		Parse(value)
	if err != nil {
		return value
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sysEnv); err != nil {
		return value
	}
	return buf.String()
}

// unsafeVars returns true when PIPE_EXPERIMENTAL_UNSAFE_VARS is set,
// disabling the vars contract so override sources can introduce new keys.
func unsafeVars() bool {
	_, ok := os.LookupEnv("PIPE_EXPERIMENTAL_UNSAFE_VARS")
	return ok
}

// ResolveVars merges pipeline vars from four sources with increasing precedence:
// YAML defaults < dot file values < system environment < CLI overrides.
// Only keys declared in yamlVars are accepted from override sources unless
// PIPE_EXPERIMENTAL_UNSAFE_VARS is set, which bypasses the contract.
func ResolveVars(yamlVars, dotFileVars, cliOverrides map[string]string) (map[string]string, []string) {
	resolved := make(map[string]string)
	var warnings []string
	sysEnv := sysEnvMap()
	unsafe := unsafeVars()

	// Build the set of declared keys (normalized to PIPE_VAR_* form).
	declared := make(map[string]bool, len(yamlVars))
	for k := range yamlVars {
		declared[VarEnvKey(k)] = true
	}

	// 1. YAML defaults (render templates against system env)
	for k, v := range yamlVars {
		resolved[VarEnvKey(k)] = renderVarValue(v, sysEnv)
	}
	// 2. Dot file values (only override declared keys unless unsafe)
	for k, v := range dotFileVars {
		envName := VarEnvKey(k)
		if unsafe || declared[envName] {
			resolved[envName] = v
		} else {
			warnings = append(warnings, fmt.Sprintf(
				"%q from dot_file has no effect — not declared in vars",
				k,
			))
		}
	}
	// 3. System env overrides (only for declared keys)
	for envName := range resolved {
		if v, ok := os.LookupEnv(envName); ok {
			resolved[envName] = v
		}
	}
	// 4. CLI overrides (only override declared keys unless unsafe)
	for k, v := range cliOverrides {
		envName := VarEnvKey(k)
		if unsafe || declared[envName] {
			resolved[envName] = v
		} else {
			warnings = append(warnings, fmt.Sprintf(
				"%q passed via CLI has no effect — not declared in vars",
				k,
			))
		}
	}
	return resolved, warnings
}

// UnmatchedEnvVarWarnings returns warnings for PIPE_VAR_* environment variables
// that are set but do not correspond to a key declared in the pipeline's vars.
// Returns nil when PIPE_EXPERIMENTAL_UNSAFE_VARS is set.
func UnmatchedEnvVarWarnings(yamlVars map[string]string) []string {
	if unsafeVars() {
		return nil
	}

	declared := make(map[string]bool, len(yamlVars))
	for k := range yamlVars {
		declared[VarEnvKey(k)] = true
	}

	var warnings []string
	for _, entry := range os.Environ() {
		k, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(k, "PIPE_VAR_") && !declared[k] {
			warnings = append(warnings, fmt.Sprintf(
				"%q is set but has no effect on this pipeline",
				k,
			))
		}
	}
	return warnings
}

// BuildEnv returns os.Environ() plus all accumulated PIPE_* vars.
func BuildEnv(pipeVars map[string]string) []string {
	env := os.Environ()
	for k, v := range pipeVars {
		env = append(env, k+"="+v)
	}
	return env
}
