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
	if got := err.Error(); !contains(got, "unexpected YAML kind") {
		t.Fatalf("expected error containing %q, got %q", "unexpected YAML kind", got)
	}
}

func TestRunField_InvalidSequenceElement(t *testing.T) {
	var r RunField
	err := yaml.Unmarshal([]byte(`[[nested]]`), &r)
	if err == nil {
		t.Fatal("expected error for nested sequence")
	}
	if got := err.Error(); !contains(got, "unexpected sequence element kind") {
		t.Fatalf("expected error containing %q, got %q", "unexpected sequence element kind", got)
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
