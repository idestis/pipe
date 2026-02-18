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
	got := ResolveVars(yaml, nil)
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
	got := ResolveVars(yaml, nil)
	if got["PIPE_VAR_NAME"] != "EnvValue" {
		t.Fatalf("expected PIPE_VAR_NAME=EnvValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIOverride(t *testing.T) {
	t.Parallel()
	yaml := map[string]string{"NAME": "Default"}
	cli := map[string]string{"NAME": "CLIValue"}
	got := ResolveVars(yaml, cli)
	if got["PIPE_VAR_NAME"] != "CLIValue" {
		t.Fatalf("expected PIPE_VAR_NAME=CLIValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIWinsOverEnv(t *testing.T) {
	t.Setenv("PIPE_VAR_NAME", "EnvValue")
	yaml := map[string]string{"NAME": "Default"}
	cli := map[string]string{"NAME": "CLIValue"}
	got := ResolveVars(yaml, cli)
	if got["PIPE_VAR_NAME"] != "CLIValue" {
		t.Fatalf("expected PIPE_VAR_NAME=CLIValue, got %q", got["PIPE_VAR_NAME"])
	}
}

func TestResolveVars_CLIIntroducesNewKey(t *testing.T) {
	t.Parallel()
	cli := map[string]string{"NEW_KEY": "newval"}
	got := ResolveVars(nil, cli)
	if got["PIPE_VAR_NEW_KEY"] != "newval" {
		t.Fatalf("expected PIPE_VAR_NEW_KEY=newval, got %q", got["PIPE_VAR_NEW_KEY"])
	}
}

func TestResolveVars_NilMaps(t *testing.T) {
	t.Parallel()
	got := ResolveVars(nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}
