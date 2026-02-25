package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDotFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseDotFile_BasicKeyValue(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "NAME=Alice\nAGE=30\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["NAME"] != "Alice" {
		t.Fatalf("expected NAME=Alice, got %q", got["NAME"])
	}
	if got["AGE"] != "30" {
		t.Fatalf("expected AGE=30, got %q", got["AGE"])
	}
}

func TestParseDotFile_CommentsAndBlankLines(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "# this is a comment\n\nFOO=bar\n\n# another comment\nBAZ=qux\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(got), got)
	}
	if got["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %q", got["FOO"])
	}
}

func TestParseDotFile_DoubleQuotedValue(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, `NAME="hello world"`)
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["NAME"] != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got["NAME"])
	}
}

func TestParseDotFile_SingleQuotedValue(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, `NAME='hello world'`)
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["NAME"] != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got["NAME"])
	}
}

func TestParseDotFile_InlineComment(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "FOO=bar # this is a comment\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %q", got["FOO"])
	}
}

func TestParseDotFile_InlineCommentNotStrippedInQuotes(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, `FOO="bar # not a comment"`)
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "bar # not a comment" {
		t.Fatalf("expected %q, got %q", "bar # not a comment", got["FOO"])
	}
}

func TestParseDotFile_EmptyValue(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "FOO=\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "" {
		t.Fatalf("expected empty value, got %q", got["FOO"])
	}
}

func TestParseDotFile_EmptyFile(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestParseDotFile_MissingFile(t *testing.T) {
	t.Parallel()
	_, _, err := ParseDotFile("/nonexistent/.env")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestParseDotFile_InvalidKey(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "VALID=ok\nINVALID KEY=value\n")
	got, warns, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["VALID"] != "ok" {
		t.Fatalf("expected VALID=ok, got %q", got["VALID"])
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "INVALID KEY") {
		t.Fatalf("expected warning about invalid key, got %v", warns)
	}
}

func TestParseDotFile_MissingEquals(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "VALID=ok\nNO_EQUALS\n")
	got, warns, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["VALID"] != "ok" {
		t.Fatalf("expected VALID=ok, got %q", got["VALID"])
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "missing '='") {
		t.Fatalf("expected warning about missing =, got %v", warns)
	}
}

func TestParseDotFile_HyphenKey(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "my-key=value\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["my-key"] != "value" {
		t.Fatalf("expected my-key=value, got %q", got["my-key"])
	}
}

func TestParseDotFile_ExportPrefix(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "export FOO=bar\nexport BAZ=qux\n")
	got, _, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %q", got["FOO"])
	}
	if got["BAZ"] != "qux" {
		t.Fatalf("expected BAZ=qux, got %q", got["BAZ"])
	}
}

func TestParseDotFile_MalformedLineSkipped(t *testing.T) {
	t.Parallel()
	path := writeDotFile(t, "GOOD=value\nBAD LINE\nINVALID KEY=x\nALSO_GOOD=ok\n")
	got, warns, err := ParseDotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(got), got)
	}
	if got["GOOD"] != "value" || got["ALSO_GOOD"] != "ok" {
		t.Fatalf("unexpected map: %v", got)
	}
	if len(warns) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(warns), warns)
	}
}
