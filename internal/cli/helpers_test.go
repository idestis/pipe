package cli

import (
	"os"
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

func TestConfirmAction_SkipTrue(t *testing.T) {
	t.Parallel()
	if !confirmAction(true, "Delete everything?") {
		t.Fatal("expected true when skip=true")
	}
}

func confirmWithStdin(t *testing.T, input string) bool {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_, _ = w.WriteString(input)
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old; r.Close() }()

	return confirmAction(false, "Proceed?")
}

func TestConfirmAction_AcceptsY(t *testing.T) {
	if confirmWithStdin(t, "y\n") != true {
		t.Fatal("expected true for 'y'")
	}
	if confirmWithStdin(t, "Y\n") != true {
		t.Fatal("expected true for 'Y'")
	}
}

func TestConfirmAction_RejectsOther(t *testing.T) {
	for _, input := range []string{"n\n", "\n", "yes\n", "no\n"} {
		if confirmWithStdin(t, input) != false {
			t.Fatalf("expected false for input %q", input)
		}
	}
}

func TestValidOwner(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase letters", "alice", true},
		{"digits", "user123", true},
		{"hyphens", "my-user", true},
		{"dots", "my.user", true},
		{"mixed valid", "alice.bob-99", true},
		{"exactly 4 chars", "abcd", true},
		{"exactly 30 chars", "abcdefghijklmnopqrstuvwxyz1234", true},
		{"too short 1 char", "a", false},
		{"too short 3 chars", "abc", false},
		{"too long 31 chars", "abcdefghijklmnopqrstuvwxyz12345", false},
		{"uppercase rejected", "Alice", false},
		{"all uppercase rejected", "ALICE", false},
		{"underscore rejected", "my_user", false},
		{"dot allowed mid-string", "a.b.c", true},
		{"starts with hyphen", "-user", false},
		{"starts with dot", ".user", false},
		{"empty string", "", false},
		{"space rejected", "my user", false},
		{"slash rejected", "my/user", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := validOwner(tt.input)
			if got != tt.want {
				t.Errorf("validOwner(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
