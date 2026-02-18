package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRunField_Scalar(t *testing.T) {
	var r RunField
	if err := yaml.Unmarshal([]byte(`"echo hi"`), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsSingle() {
		t.Fatal("expected IsSingle() == true")
	}
	if r.Single != "echo hi" {
		t.Fatalf("expected %q, got %q", "echo hi", r.Single)
	}
}

func TestRunField_StringSequence(t *testing.T) {
	var r RunField
	if err := yaml.Unmarshal([]byte(`["a","b","c"]`), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsStrings() {
		t.Fatal("expected IsStrings() == true")
	}
	if len(r.Strings) != 3 {
		t.Fatalf("expected 3 strings, got %d", len(r.Strings))
	}
}

func TestRunField_SubRunSequence(t *testing.T) {
	input := `
- id: a
  run: "echo a"
  sensitive: true
- id: b
  run: "echo b"
`
	var r RunField
	if err := yaml.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsSubRuns() {
		t.Fatal("expected IsSubRuns() == true")
	}
	if len(r.SubRuns) != 2 {
		t.Fatalf("expected 2 sub-runs, got %d", len(r.SubRuns))
	}
	if r.SubRuns[0].ID != "a" || !r.SubRuns[0].Sensitive {
		t.Fatalf("sub-run 0: expected id=a sensitive=true, got id=%q sensitive=%v", r.SubRuns[0].ID, r.SubRuns[0].Sensitive)
	}
	if r.SubRuns[1].ID != "b" || r.SubRuns[1].Sensitive {
		t.Fatalf("sub-run 1: expected id=b sensitive=false, got id=%q sensitive=%v", r.SubRuns[1].ID, r.SubRuns[1].Sensitive)
	}
}

func TestRunField_EmptySequence(t *testing.T) {
	var r RunField
	err := yaml.Unmarshal([]byte(`[]`), &r)
	if err == nil {
		t.Fatal("expected error for empty sequence")
	}
	if got := err.Error(); !contains(got, "empty sequence") {
		t.Fatalf("expected error containing %q, got %q", "empty sequence", got)
	}
}

func TestRunField_InvalidTopLevelKind(t *testing.T) {
	var r RunField
	err := yaml.Unmarshal([]byte(`{key: val}`), &r)
	if err == nil {
		t.Fatal("expected error for mapping")
	}
	if got := err.Error(); !contains(got, "must be a string command or a list of commands") {
		t.Fatalf("expected error containing %q, got %q", "must be a string command or a list of commands", got)
	}
}

func TestRunField_InvalidSequenceElement(t *testing.T) {
	var r RunField
	err := yaml.Unmarshal([]byte(`[[nested]]`), &r)
	if err == nil {
		t.Fatal("expected error for nested sequence")
	}
	if got := err.Error(); !contains(got, "each list item must be a string or a mapping") {
		t.Fatalf("expected error containing %q, got %q", "each list item must be a string or a mapping", got)
	}
}

func TestPipeline_FullUnmarshal(t *testing.T) {
	input := `
name: full
description: "full pipeline"
steps:
  - id: single
    run: "echo hello"
  - id: strings
    run: ["a", "b"]
  - id: subs
    run:
      - id: x
        run: "echo x"
        sensitive: true
      - id: y
        run: "echo y"
  - id: retried
    run: "echo retry"
    retry: 3
    sensitive: true
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "full" {
		t.Fatalf("expected name %q, got %q", "full", p.Name)
	}
	if p.Description != "full pipeline" {
		t.Fatalf("expected description %q, got %q", "full pipeline", p.Description)
	}
	if len(p.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(p.Steps))
	}
	if !p.Steps[0].Run.IsSingle() {
		t.Fatal("step 0: expected IsSingle()")
	}
	if !p.Steps[1].Run.IsStrings() {
		t.Fatal("step 1: expected IsStrings()")
	}
	if !p.Steps[2].Run.IsSubRuns() {
		t.Fatal("step 2: expected IsSubRuns()")
	}
	if p.Steps[3].Retry != 3 {
		t.Fatalf("step 3: expected retry=3, got %d", p.Steps[3].Retry)
	}
	if !p.Steps[3].Sensitive {
		t.Fatal("step 3: expected sensitive=true")
	}
}

func TestPipeline_DescriptionOptional(t *testing.T) {
	input := `
name: nodesc
steps:
  - id: a
    run: "echo a"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Description != "" {
		t.Fatalf("expected empty description, got %q", p.Description)
	}
}

func TestPipeline_EmptySteps(t *testing.T) {
	input := `
name: empty
steps: []
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps) != 0 {
		t.Fatalf("expected 0 steps, got %d", len(p.Steps))
	}
}

func TestPipeline_NoStepsKey(t *testing.T) {
	input := `name: minimal`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Steps != nil {
		t.Fatalf("expected nil steps, got %v", p.Steps)
	}
}

func TestHelpers_ZeroValue(t *testing.T) {
	var r RunField
	if r.IsSingle() {
		t.Fatal("zero RunField: IsSingle() should be false")
	}
	if r.IsStrings() {
		t.Fatal("zero RunField: IsStrings() should be false")
	}
	if r.IsSubRuns() {
		t.Fatal("zero RunField: IsSubRuns() should be false")
	}
}

func TestPipeline_FullUnmarshalWithCached(t *testing.T) {
	input := `
name: cached-pipeline
description: "pipeline with caching"
steps:
  - id: sso-login
    run: "aws sso login"
    cached: true
  - id: build
    run: "npm run build"
    cached:
      expireAfter: "1h"
  - id: deploy
    run: "deploy --prod"
    cached:
      expireAfter: "18:10 UTC"
  - id: no-cache
    run: "echo hello"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(p.Steps))
	}
	// cached: true
	if !p.Steps[0].Cached.Enabled {
		t.Fatal("step 0: expected Cached.Enabled == true")
	}
	if p.Steps[0].Cached.ExpireAfter != "" {
		t.Fatalf("step 0: expected empty ExpireAfter, got %q", p.Steps[0].Cached.ExpireAfter)
	}
	// cached with duration
	if !p.Steps[1].Cached.Enabled {
		t.Fatal("step 1: expected Cached.Enabled == true")
	}
	if p.Steps[1].Cached.ExpireAfter != "1h" {
		t.Fatalf("step 1: expected ExpireAfter %q, got %q", "1h", p.Steps[1].Cached.ExpireAfter)
	}
	// cached with absolute time
	if !p.Steps[2].Cached.Enabled {
		t.Fatal("step 2: expected Cached.Enabled == true")
	}
	if p.Steps[2].Cached.ExpireAfter != "18:10 UTC" {
		t.Fatalf("step 2: expected ExpireAfter %q, got %q", "18:10 UTC", p.Steps[2].Cached.ExpireAfter)
	}
	// no cached field
	if p.Steps[3].Cached.Enabled {
		t.Fatal("step 3: expected Cached.Enabled == false")
	}
}

func TestPipeline_WithVars(t *testing.T) {
	input := `
name: with-vars
vars:
  GREETING: "Hello"
  DB_HOST: "localhost"
steps:
  - id: greet
    run: "echo $PIPE_VAR_GREETING"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(p.Vars))
	}
	if p.Vars["GREETING"] != "Hello" {
		t.Fatalf("expected GREETING=%q, got %q", "Hello", p.Vars["GREETING"])
	}
	if p.Vars["DB_HOST"] != "localhost" {
		t.Fatalf("expected DB_HOST=%q, got %q", "localhost", p.Vars["DB_HOST"])
	}
}

func TestPipeline_WithoutVars(t *testing.T) {
	input := `
name: no-vars
steps:
  - id: hello
    run: "echo hi"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Vars != nil {
		t.Fatalf("expected nil vars, got %v", p.Vars)
	}
}

func TestPipeline_EmptyVars(t *testing.T) {
	input := `
name: empty-vars
vars: {}
steps:
  - id: hello
    run: "echo hi"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Vars) != 0 {
		t.Fatalf("expected 0 vars, got %d", len(p.Vars))
	}
}

func TestDependsOnField_Scalar(t *testing.T) {
	input := `
name: test
steps:
  - id: a
    run: "echo a"
  - id: b
    run: "echo b"
    depends_on: "a"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps[1].DependsOn.Steps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(p.Steps[1].DependsOn.Steps))
	}
	if p.Steps[1].DependsOn.Steps[0] != "a" {
		t.Fatalf("expected dependency %q, got %q", "a", p.Steps[1].DependsOn.Steps[0])
	}
}

func TestDependsOnField_Sequence(t *testing.T) {
	input := `
name: test
steps:
  - id: a
    run: "echo a"
  - id: b
    run: "echo b"
  - id: c
    run: "echo c"
    depends_on: ["a", "b"]
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps[2].DependsOn.Steps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(p.Steps[2].DependsOn.Steps))
	}
	if p.Steps[2].DependsOn.Steps[0] != "a" || p.Steps[2].DependsOn.Steps[1] != "b" {
		t.Fatalf("unexpected dependencies: %v", p.Steps[2].DependsOn.Steps)
	}
}

func TestDependsOnField_Empty(t *testing.T) {
	input := `
name: test
steps:
  - id: a
    run: "echo a"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps[0].DependsOn.Steps) != 0 {
		t.Fatalf("expected 0 dependencies, got %d", len(p.Steps[0].DependsOn.Steps))
	}
}

func TestDependsOnField_InvalidType(t *testing.T) {
	input := `
name: test
steps:
  - id: a
    run: "echo a"
    depends_on:
      key: val
`
	var p Pipeline
	err := yaml.Unmarshal([]byte(input), &p)
	if err == nil {
		t.Fatal("expected error for mapping depends_on")
	}
}

// contains is a tiny helper to avoid importing strings in tests.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
