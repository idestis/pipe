package runner

import (
	"os"
	"strings"
	"testing"
)

func TestEnvKey_Single(t *testing.T) {
	t.Parallel()
	if got := EnvKey("build"); got != "PIPE_BUILD" {
		t.Fatalf("expected PIPE_BUILD, got %s", got)
	}
}

func TestEnvKey_Multi(t *testing.T) {
	t.Parallel()
	if got := EnvKey("deploy", "east"); got != "PIPE_DEPLOY_EAST" {
		t.Fatalf("expected PIPE_DEPLOY_EAST, got %s", got)
	}
}

func TestEnvKey_Hyphens(t *testing.T) {
	t.Parallel()
	if got := EnvKey("my-step", "sub-run"); got != "PIPE_MY_STEP_SUB_RUN" {
		t.Fatalf("expected PIPE_MY_STEP_SUB_RUN, got %s", got)
	}
}

func TestBuildEnv_IncludesPipeVars(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"PIPE_FOO": "bar"}
	env := BuildEnv(vars)

	// Should contain at least one OS env var (PATH is always present).
	hasPath := false
	hasPipeFoo := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
		}
		if e == "PIPE_FOO=bar" {
			hasPipeFoo = true
		}
	}
	if !hasPath {
		t.Fatal("expected PATH in env")
	}
	if !hasPipeFoo {
		t.Fatal("expected PIPE_FOO=bar in env")
	}

	// Total should be os.Environ() + 1
	if len(env) != len(os.Environ())+1 {
		t.Fatalf("expected %d entries, got %d", len(os.Environ())+1, len(env))
	}
}

func TestBuildEnv_EmptyMap(t *testing.T) {
	t.Parallel()
	env := BuildEnv(map[string]string{})
	if len(env) != len(os.Environ()) {
		t.Fatalf("expected %d entries, got %d", len(os.Environ()), len(env))
	}
}

func TestVarEnvKey_Lowercase(t *testing.T) {
	t.Parallel()
	if got := VarEnvKey("greeting"); got != "PIPE_VAR_GREETING" {
		t.Fatalf("expected PIPE_VAR_GREETING, got %s", got)
	}
}

func TestVarEnvKey_MixedCase(t *testing.T) {
	t.Parallel()
	if got := VarEnvKey("DbHost"); got != "PIPE_VAR_DBHOST" {
		t.Fatalf("expected PIPE_VAR_DBHOST, got %s", got)
	}
}

func TestVarEnvKey_Hyphens(t *testing.T) {
	t.Parallel()
	if got := VarEnvKey("db-host"); got != "PIPE_VAR_DB_HOST" {
		t.Fatalf("expected PIPE_VAR_DB_HOST, got %s", got)
	}
}

func TestVarEnvKey_Uppercase(t *testing.T) {
	t.Parallel()
	if got := VarEnvKey("RECIPIENT"); got != "PIPE_VAR_RECIPIENT" {
		t.Fatalf("expected PIPE_VAR_RECIPIENT, got %s", got)
	}
}

func TestResolveVars_YAMLOnly(t *testing.T) {
	t.Parallel()
	yaml := map[string]string{"GREETING": "Hello", "NAME": "World"}
	got, _ := ResolveVars(yaml, nil, nil)
	if got["PIPE_VAR_GREETING"] != "Hello" {
		t.Fatalf("expected PIPE_VAR_GREETING=Hello, got %q", got["PIPE_VAR_GREETING"])
	}
	if got["PIPE_VAR_NAME"] != "World" {
		t.Fatalf("expected PIPE_VAR_NAME=World, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_EnvOverride(t *testing.T) {
	t.Setenv("PIPE_VAR_NAME", "EnvValue")
	yaml := map[string]string{"NAME": "Default"}
	got, _ := ResolveVars(yaml, nil, nil)
	if got["PIPE_VAR_NAME"] != "EnvValue" {
		t.Fatalf("expected PIPE_VAR_NAME=EnvValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIOverride(t *testing.T) {
	t.Parallel()
	yaml := map[string]string{"NAME": "Default"}
	cli := map[string]string{"NAME": "CLIValue"}
	got, _ := ResolveVars(yaml, nil, cli)
	if got["PIPE_VAR_NAME"] != "CLIValue" {
		t.Fatalf("expected PIPE_VAR_NAME=CLIValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIWinsOverEnv(t *testing.T) {
	t.Setenv("PIPE_VAR_NAME", "EnvValue")
	yaml := map[string]string{"NAME": "Default"}
	cli := map[string]string{"NAME": "CLIValue"}
	got, _ := ResolveVars(yaml, nil, cli)
	if got["PIPE_VAR_NAME"] != "CLIValue" {
		t.Fatalf("expected PIPE_VAR_NAME=CLIValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIUnknownKeyWarns(t *testing.T) {
	t.Parallel()
	yaml := map[string]string{"NAME": "default"}
	cli := map[string]string{"NEW_KEY": "newval"}
	got, warns := ResolveVars(yaml, nil, cli)
	if _, ok := got["PIPE_VAR_NEW_KEY"]; ok {
		t.Fatal("undeclared CLI key should not be in resolved map")
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "NEW_KEY") || !strings.Contains(warns[0], "CLI") {
		t.Fatalf("expected warning about undeclared CLI key, got %v", warns)
	}
}

func TestResolveVars_NilMaps(t *testing.T) {
	t.Parallel()
	got, warns := ResolveVars(nil, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
	if len(warns) != 0 {
		t.Fatalf("expected no warnings, got %v", warns)
	}
}

// --- renderVarValue tests ---

func TestRenderVarValue_PlainString(t *testing.T) {
	t.Parallel()
	got := renderVarValue("hello world", map[string]string{})
	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}
}

func TestRenderVarValue_Default(t *testing.T) {
	t.Parallel()
	got := renderVarValue(`{{ .MISSING | default "fallback" }}`, map[string]string{})
	if got != "fallback" {
		t.Fatalf("expected %q, got %q", "fallback", got)
	}
}

func TestRenderVarValue_EnvRef(t *testing.T) {
	t.Parallel()
	env := map[string]string{"HOME": "/home/test"}
	got := renderVarValue("{{ .HOME }}", env)
	if got != "/home/test" {
		t.Fatalf("expected %q, got %q", "/home/test", got)
	}
}

func TestRenderVarValue_InvalidTemplate(t *testing.T) {
	t.Parallel()
	raw := "{{ .foo | bad }}"
	got := renderVarValue(raw, map[string]string{"foo": "x"})
	if got != raw {
		t.Fatalf("expected original %q, got %q", raw, got)
	}
}

func TestResolveVars_RenderedDefault(t *testing.T) {
	t.Parallel()
	yaml := map[string]string{
		"WHO": `{{ .USER | default "Anon" }}`,
	}
	got, _ := ResolveVars(yaml, nil, nil)
	val := got["PIPE_VAR_WHO"]
	// USER may or may not be set; either way the template should resolve.
	if val == "" || strings.Contains(val, "{{") {
		t.Fatalf("expected rendered value, got %q", val)
	}
}

// --- dot file integration tests ---

func TestResolveVars_DotFileOverridesYAML(t *testing.T) {
	t.Parallel()
	yamlVars := map[string]string{"NAME": "yaml-default"}
	dotVars := map[string]string{"NAME": "dotfile-value"}
	got, warns := ResolveVars(yamlVars, dotVars, nil)
	if got["PIPE_VAR_NAME"] != "dotfile-value" {
		t.Fatalf("expected PIPE_VAR_NAME=dotfile-value, got %q", got["PIPE_VAR_NAME"])
	}
	if len(warns) != 0 {
		t.Fatalf("expected no warnings, got %v", warns)
	}
}

func TestResolveVars_EnvOverridesDotFile(t *testing.T) {
	t.Setenv("PIPE_VAR_NAME", "env-value")
	yamlVars := map[string]string{"NAME": "yaml-default"}
	dotVars := map[string]string{"NAME": "dotfile-value"}
	got, _ := ResolveVars(yamlVars, dotVars, nil)
	if got["PIPE_VAR_NAME"] != "env-value" {
		t.Fatalf("expected PIPE_VAR_NAME=env-value, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIOverridesDotFile(t *testing.T) {
	t.Parallel()
	yamlVars := map[string]string{"NAME": "yaml-default"}
	dotVars := map[string]string{"NAME": "dotfile-value"}
	cli := map[string]string{"NAME": "cli-value"}
	got, _ := ResolveVars(yamlVars, dotVars, cli)
	if got["PIPE_VAR_NAME"] != "cli-value" {
		t.Fatalf("expected PIPE_VAR_NAME=cli-value, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_DotFileUnknownKeyWarns(t *testing.T) {
	t.Parallel()
	yamlVars := map[string]string{"NAME": "default"}
	dotVars := map[string]string{"NEW_KEY": "new-value"}
	got, warns := ResolveVars(yamlVars, dotVars, nil)
	if _, ok := got["PIPE_VAR_NEW_KEY"]; ok {
		t.Fatal("undeclared dot_file key should not be in resolved map")
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "NEW_KEY") || !strings.Contains(warns[0], "dot_file") {
		t.Fatalf("expected warning about undeclared dot_file key, got %v", warns)
	}
}

func TestResolveVars_FullPrecedenceChain(t *testing.T) {
	t.Setenv("PIPE_VAR_B", "env-b")
	yamlVars := map[string]string{"A": "yaml-a", "B": "yaml-b", "C": "yaml-c", "D": "yaml-d"}
	dotVars := map[string]string{"B": "dot-b", "C": "dot-c", "D": "dot-d"}
	cli := map[string]string{"D": "cli-d"}
	got, warns := ResolveVars(yamlVars, dotVars, cli)
	if len(warns) != 0 {
		t.Fatalf("expected no warnings (all keys declared), got %v", warns)
	}
	// A: only YAML
	if got["PIPE_VAR_A"] != "yaml-a" {
		t.Fatalf("A: expected yaml-a, got %q", got["PIPE_VAR_A"])
	}
	// B: YAML < dot < env (env wins)
	if got["PIPE_VAR_B"] != "env-b" {
		t.Fatalf("B: expected env-b, got %q", got["PIPE_VAR_B"])
	}
	// C: YAML < dot (dot wins, no env or CLI)
	if got["PIPE_VAR_C"] != "dot-c" {
		t.Fatalf("C: expected dot-c, got %q", got["PIPE_VAR_C"])
	}
	// D: YAML < dot < CLI (CLI wins)
	if got["PIPE_VAR_D"] != "cli-d" {
		t.Fatalf("D: expected cli-d, got %q", got["PIPE_VAR_D"])
	}
}

// --- UnmatchedEnvVarWarnings tests ---

func TestUnmatchedEnvVarWarnings_MatchedKey(t *testing.T) {
	t.Setenv("PIPE_VAR_FOO", "bar")
	yamlVars := map[string]string{"FOO": "default"}
	warns := UnmatchedEnvVarWarnings(yamlVars)
	for _, w := range warns {
		if strings.Contains(w, "PIPE_VAR_FOO") {
			t.Fatalf("should not warn about declared key, got %q", w)
		}
	}
}

func TestUnmatchedEnvVarWarnings_UnmatchedKey(t *testing.T) {
	t.Setenv("PIPE_VAR_NONAME", "value")
	yamlVars := map[string]string{"FOO": "default"}
	warns := UnmatchedEnvVarWarnings(yamlVars)
	found := false
	for _, w := range warns {
		if strings.Contains(w, "PIPE_VAR_NONAME") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning about PIPE_VAR_NONAME, got %v", warns)
	}
}

// --- PIPE_EXPERIMENTAL_UNSAFE_VARS tests ---

func TestResolveVars_UnsafeVarsCLIIntroducesNewKey(t *testing.T) {
	t.Setenv("PIPE_EXPERIMENTAL_UNSAFE_VARS", "1")
	yaml := map[string]string{"NAME": "default"}
	cli := map[string]string{"NEW_KEY": "newval"}
	got, warns := ResolveVars(yaml, nil, cli)
	if got["PIPE_VAR_NEW_KEY"] != "newval" {
		t.Fatalf("expected PIPE_VAR_NEW_KEY=newval, got %q", got["PIPE_VAR_NEW_KEY"])
	}
	if len(warns) != 0 {
		t.Fatalf("expected no warnings in unsafe mode, got %v", warns)
	}
}

func TestResolveVars_UnsafeVarsDotFileIntroducesNewKey(t *testing.T) {
	t.Setenv("PIPE_EXPERIMENTAL_UNSAFE_VARS", "1")
	yaml := map[string]string{"NAME": "default"}
	dotVars := map[string]string{"NEW_KEY": "new-value"}
	got, warns := ResolveVars(yaml, dotVars, nil)
	if got["PIPE_VAR_NEW_KEY"] != "new-value" {
		t.Fatalf("expected PIPE_VAR_NEW_KEY=new-value, got %q", got["PIPE_VAR_NEW_KEY"])
	}
	if len(warns) != 0 {
		t.Fatalf("expected no warnings in unsafe mode, got %v", warns)
	}
}

func TestUnmatchedEnvVarWarnings_UnsafeSkipsWarnings(t *testing.T) {
	t.Setenv("PIPE_EXPERIMENTAL_UNSAFE_VARS", "1")
	t.Setenv("PIPE_VAR_NONAME", "value")
	yamlVars := map[string]string{"FOO": "default"}
	warns := UnmatchedEnvVarWarnings(yamlVars)
	if len(warns) != 0 {
		t.Fatalf("expected no warnings in unsafe mode, got %v", warns)
	}
}
