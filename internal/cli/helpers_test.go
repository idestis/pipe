package cli

import (
	"strings"
	"testing"
)

func TestParseVarOverrides_Valid(t *testing.T) {
	t.Parallel()
	got, err := parseVarOverrides([]string{"NAME=Claude", "HOST=localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["NAME"] != "Claude" {
		t.Fatalf("expected NAME=Claude, got %q", got["NAME"])
	}
	if got["HOST"] != "localhost" {
		t.Fatalf("expected HOST=localhost, got %q", got["HOST"])
	}
}

func TestParseVarOverrides_Empty(t *testing.T) {
	t.Parallel()
	got, err := parseVarOverrides(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestParseVarOverrides_MissingEquals(t *testing.T) {
	t.Parallel()
	_, err := parseVarOverrides([]string{"NOEQUALS"})
	if err == nil {
		t.Fatal("expected error for missing =")
	}
	if !strings.Contains(err.Error(), "expected KEY=value") {
		t.Fatalf("expected error containing %q, got %q", "expected KEY=value", err.Error())
	}
}

func TestParseVarOverrides_EmptyKey(t *testing.T) {
	t.Parallel()
	_, err := parseVarOverrides([]string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "invalid variable key") {
		t.Fatalf("expected error containing %q, got %q", "invalid variable key", err.Error())
	}
}

func TestParseVarOverrides_ValueWithEquals(t *testing.T) {
	t.Parallel()
	got, err := parseVarOverrides([]string{"DSN=host=db user=admin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["DSN"] != "host=db user=admin" {
		t.Fatalf("expected DSN=%q, got %q", "host=db user=admin", got["DSN"])
	}
}

func TestParseVarOverrides_EmptyValue(t *testing.T) {
	t.Parallel()
	got, err := parseVarOverrides([]string{"KEY="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["KEY"] != "" {
		t.Fatalf("expected KEY=%q, got %q", "", got["KEY"])
	}
}

func TestParseVarOverrides_InvalidKeyChars(t *testing.T) {
	t.Parallel()
	_, err := parseVarOverrides([]string{"my.var=value"})
	if err == nil {
		t.Fatal("expected error for invalid key chars")
	}
	if !strings.Contains(err.Error(), "invalid variable key") {
		t.Fatalf("expected error containing %q, got %q", "invalid variable key", err.Error())
	}
}
