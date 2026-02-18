package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idestis/pipe/internal/config"
)

// overrideFilesDir points config.FilesDir at a temp directory for the test
// and restores the original value when the test finishes.
func overrideFilesDir(t *testing.T) string {
	t.Helper()
	orig := config.FilesDir
	tmp := t.TempDir()
	config.FilesDir = tmp
	t.Cleanup(func() { config.FilesDir = orig })
	return tmp
}

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPipeline_Valid(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "deploy", `
name: deploy
description: "deployment pipeline"
steps:
  - id: build
    run: "echo build"
`)
	p, err := LoadPipeline("deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "deploy" {
		t.Fatalf("expected name %q, got %q", "deploy", p.Name)
	}
	if p.Description != "deployment pipeline" {
		t.Fatalf("expected description %q, got %q", "deployment pipeline", p.Description)
	}
	if len(p.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(p.Steps))
	}
}

func TestLoadPipeline_NameDefaultsToFilename(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "myfile", `
steps:
  - id: hello
    run: "echo hi"
`)
	p, err := LoadPipeline("myfile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "myfile" {
		t.Fatalf("expected name %q, got %q", "myfile", p.Name)
	}
}

func TestLoadPipeline_FileNotFound(t *testing.T) {
	overrideFilesDir(t)
	_, err := LoadPipeline("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading pipeline") {
		t.Fatalf("expected error containing %q, got %q", "reading pipeline", err.Error())
	}
}

func TestLoadPipeline_InvalidYAML(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "bad", `{{{invalid`)
	_, err := LoadPipeline("bad")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing pipeline") {
		t.Fatalf("expected error containing %q, got %q", "parsing pipeline", err.Error())
	}
}

func TestValidate_MissingID(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "noid", `
steps:
  - run: "echo hi"
`)
	_, err := LoadPipeline("noid")
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("expected error containing %q, got %q", "missing id", err.Error())
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "dupid", `
steps:
  - id: same
    run: "echo a"
  - id: same
    run: "echo b"
`)
	_, err := LoadPipeline("dupid")
	if err == nil {
		t.Fatal("expected error for duplicate id")
	}
	if !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("expected error containing %q, got %q", "duplicate id", err.Error())
	}
}

func TestValidate_MissingRun(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "norun", `
steps:
  - id: empty
`)
	_, err := LoadPipeline("norun")
	if err == nil {
		t.Fatal("expected error for missing run field")
	}
	if !strings.Contains(err.Error(), "missing run field") {
		t.Fatalf("expected error containing %q, got %q", "missing run field", err.Error())
	}
}

func TestValidate_Valid(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "ok", `
steps:
  - id: a
    run: "echo a"
  - id: b
    run: ["x", "y"]
`)
	_, err := LoadPipeline("ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPipelines_Multiple(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "beta", `
name: beta
description: "second"
steps:
  - id: b
    run: "echo b"
`)
	writeYAML(t, dir, "alpha", `
name: alpha
description: "first"
steps:
  - id: a
    run: "echo a"
`)
	infos, err := ListPipelines()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(infos))
	}
	if infos[0].Name != "alpha" {
		t.Fatalf("expected first pipeline %q, got %q", "alpha", infos[0].Name)
	}
	if infos[1].Name != "beta" {
		t.Fatalf("expected second pipeline %q, got %q", "beta", infos[1].Name)
	}
	if infos[0].Description != "first" {
		t.Fatalf("expected description %q, got %q", "first", infos[0].Description)
	}
}

func TestListPipelines_Empty(t *testing.T) {
	overrideFilesDir(t)
	infos, err := ListPipelines()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Fatalf("expected 0 pipelines, got %d", len(infos))
	}
}

func TestListPipelines_NameFallback(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "noname", `
steps:
  - id: a
    run: "echo a"
`)
	infos, err := ListPipelines()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(infos))
	}
	if infos[0].Name != "noname" {
		t.Fatalf("expected name %q, got %q", "noname", infos[0].Name)
	}
}

func TestValidatePipeline_Valid(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "good", `
name: good
steps:
  - id: a
    run: "echo a"
`)
	if err := ValidatePipeline("good"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWarnings_CachedAndSensitive(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "warn-cached", `
name: warn-cached
steps:
  - id: secret
    run: "vault read token"
    sensitive: true
    cached: true
`)
	p, err := LoadPipeline("warn-cached")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	warns := Warnings(p)
	if len(warns) == 0 {
		t.Fatal("expected warnings for cached + sensitive")
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "cached + sensitive") && strings.Contains(w, "PIPE_SECRET") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning about cached + sensitive with env var, got: %v", warns)
	}
}

func TestWarnings_SensitiveVarReferenced(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "warn-ref", `
name: warn-ref
steps:
  - id: get-token
    run: "vault read -field=token secret/deploy"
    sensitive: true
  - id: deploy
    run: "curl -H \"Authorization: $PIPE_GET_TOKEN\" https://api.example.com"
`)
	p, err := LoadPipeline("warn-ref")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	warns := Warnings(p)
	if len(warns) == 0 {
		t.Fatal("expected warnings for sensitive var reference")
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "PIPE_GET_TOKEN") && strings.Contains(w, "re-execute on resume") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning about sensitive step re-execution, got: %v", warns)
	}
}

func TestWarnings_SensitiveVarBracesSyntax(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "warn-braces", `
name: warn-braces
steps:
  - id: get-token
    run: "vault read token"
    sensitive: true
  - id: deploy
    run: "echo ${PIPE_GET_TOKEN}"
`)
	p, err := LoadPipeline("warn-braces")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	warns := Warnings(p)
	found := false
	for _, w := range warns {
		if strings.Contains(w, "PIPE_GET_TOKEN") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning for ${} syntax reference, got: %v", warns)
	}
}

func TestWarnings_NoWarningsForCleanPipeline(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "clean", `
name: clean
steps:
  - id: build
    run: "echo build"
    cached: true
  - id: deploy
    run: "echo deploy"
`)
	p, err := LoadPipeline("clean")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	warns := Warnings(p)
	if len(warns) != 0 {
		t.Fatalf("expected no warnings, got: %v", warns)
	}
}

func TestWarnings_SensitiveSubRunVarReferenced(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "warn-subrun", `
name: warn-subrun
steps:
  - id: fetch
    run:
      - id: secret-api
        run: "curl -s https://secret.api/token"
        sensitive: true
    sensitive: true
  - id: use-it
    run: "echo $PIPE_FETCH_SECRET_API"
`)
	p, err := LoadPipeline("warn-subrun")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	warns := Warnings(p)
	found := false
	for _, w := range warns {
		if strings.Contains(w, "PIPE_FETCH_SECRET_API") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning for sensitive sub-run var reference, got: %v", warns)
	}
}

func TestValidate_ValidVarKeys(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "goodvars", `
name: goodvars
vars:
  GREETING: "Hello"
  db-host: "localhost"
  my_var_2: "val"
steps:
  - id: a
    run: "echo hi"
`)
	_, err := LoadPipeline("goodvars")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyVarKey(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "emptyvar", `
name: emptyvar
vars:
  "": "value"
steps:
  - id: a
    run: "echo hi"
`)
	_, err := LoadPipeline("emptyvar")
	if err == nil {
		t.Fatal("expected error for empty var key")
	}
	if !strings.Contains(err.Error(), "invalid var key") {
		t.Fatalf("expected error containing %q, got %q", "invalid var key", err.Error())
	}
}

func TestValidate_InvalidVarKeyChars(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "badvar", `
name: badvar
vars:
  "my.var": "value"
steps:
  - id: a
    run: "echo hi"
`)
	_, err := LoadPipeline("badvar")
	if err == nil {
		t.Fatal("expected error for invalid var key chars")
	}
	if !strings.Contains(err.Error(), "invalid var key") {
		t.Fatalf("expected error containing %q, got %q", "invalid var key", err.Error())
	}
}

func TestValidatePipeline_Invalid(t *testing.T) {
	dir := overrideFilesDir(t)
	writeYAML(t, dir, "bad", `
name: bad
steps:
  - run: "echo missing id"
`)
	err := ValidatePipeline("bad")
	if err == nil {
		t.Fatal("expected error for invalid pipeline")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("expected error containing %q, got %q", "missing id", err.Error())
	}
}
