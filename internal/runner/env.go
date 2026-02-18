package runner

import (
	"os"
	"strings"
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

// ResolveVars merges pipeline vars from three sources with increasing precedence:
// YAML defaults < system environment < CLI overrides.
func ResolveVars(yamlVars map[string]string, cliOverrides map[string]string) map[string]string {
	resolved := make(map[string]string)
	// 1. YAML defaults
	for k, v := range yamlVars {
		resolved[VarEnvKey(k)] = v
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
