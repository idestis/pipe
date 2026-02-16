package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/destis/pipe/internal/config"
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
