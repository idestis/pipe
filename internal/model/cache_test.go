package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCacheField_BoolTrue(t *testing.T) {
	var c CacheField
	if err := yaml.Unmarshal([]byte(`true`), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Enabled {
		t.Fatal("expected Enabled == true")
	}
	if c.ExpireAfter != "" {
		t.Fatalf("expected empty ExpireAfter, got %q", c.ExpireAfter)
	}
}

func TestCacheField_BoolFalse(t *testing.T) {
	var c CacheField
	if err := yaml.Unmarshal([]byte(`false`), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Enabled {
		t.Fatal("expected Enabled == false")
	}
}

func TestCacheField_MappingWithExpiry(t *testing.T) {
	input := `expireAfter: "1h"`
	var c CacheField
	if err := yaml.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Enabled {
		t.Fatal("expected Enabled == true")
	}
	if c.ExpireAfter != "1h" {
		t.Fatalf("expected ExpireAfter %q, got %q", "1h", c.ExpireAfter)
	}
}

func TestCacheField_MappingAbsoluteTime(t *testing.T) {
	input := `expireAfter: "18:10 UTC"`
	var c CacheField
	if err := yaml.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Enabled {
		t.Fatal("expected Enabled == true")
	}
	if c.ExpireAfter != "18:10 UTC" {
		t.Fatalf("expected ExpireAfter %q, got %q", "18:10 UTC", c.ExpireAfter)
	}
}

func TestCacheField_InvalidScalar(t *testing.T) {
	var c CacheField
	err := yaml.Unmarshal([]byte(`"notabool"`), &c)
	if err == nil {
		t.Fatal("expected error for invalid scalar")
	}
}

func TestCacheField_InvalidKind(t *testing.T) {
	var c CacheField
	err := yaml.Unmarshal([]byte(`["a","b"]`), &c)
	if err == nil {
		t.Fatal("expected error for sequence")
	}
}

func TestCacheField_ZeroValue(t *testing.T) {
	var c CacheField
	if c.Enabled {
		t.Fatal("zero CacheField: Enabled should be false")
	}
	if c.ExpireAfter != "" {
		t.Fatal("zero CacheField: ExpireAfter should be empty")
	}
}

func TestCacheField_InStep(t *testing.T) {
	input := `
id: build
run: "npm run build"
cache: true
`
	var s Step
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Cached.Enabled {
		t.Fatal("expected Cached.Enabled == true")
	}
}

func TestCacheField_InStepWithExpiry(t *testing.T) {
	input := `
id: build
run: "npm run build"
cache:
  expireAfter: "30m"
`
	var s Step
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Cached.Enabled {
		t.Fatal("expected Cached.Enabled == true")
	}
	if s.Cached.ExpireAfter != "30m" {
		t.Fatalf("expected ExpireAfter %q, got %q", "30m", s.Cached.ExpireAfter)
	}
}
