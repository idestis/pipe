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

// ResolveVars merges pipeline vars from three sources with increasing precedence:
// YAML defaults < system environment < CLI overrides.
func ResolveVars(yamlVars map[string]string, cliOverrides map[string]string) map[string]string {
	resolved := make(map[string]string)
	sysEnv := sysEnvMap()
	// 1. YAML defaults (render templates against system env)
	for k, v := range yamlVars {
		resolved[VarEnvKey(k)] = renderVarValue(v, sysEnv)
	}
	// 2. System env overrides (only for declared keys)
	for envName := range resolved {
		if v, ok := os.LookupEnv(envName); ok {
			resolved[envName] = v
		}
	}
	// 3. CLI overrides (highest precedence, can introduce new keys)
	for k, v := range cliOverrides {
		resolved[VarEnvKey(k)] = v
	}
	return resolved
}

// BuildEnv returns os.Environ() plus all accumulated PIPE_* vars.
func BuildEnv(pipeVars map[string]string) []string {
	env := os.Environ()
	for k, v := range pipeVars {
		env = append(env, k+"="+v)
	}
	return env
}
