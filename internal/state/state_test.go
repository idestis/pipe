package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/getpipe-dev/pipe/internal/config"
)

// overrideStateDir points config.StateDir at a temp directory for the test
// and restores the original value when the test finishes.
func overrideStateDir(t *testing.T) string {
	t.Helper()
	orig := config.StateDir
	tmp := t.TempDir()
	config.StateDir = tmp
	t.Cleanup(func() { config.StateDir = orig })
	return tmp
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUID_Format(t *testing.T) {
	for range 100 {
		id := NewUUID()
		if !uuidRe.MatchString(id) {
			t.Fatalf("UUID %q does not match expected format", id)
		}
	}
}

func TestNewUUID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for range 1000 {
		id := NewUUID()
		if seen[id] {
			t.Fatalf("duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	tmp := overrideStateDir(t)
	if err := os.MkdirAll(filepath.Join(tmp, "test-pipe"), 0o755); err != nil {
		t.Fatal(err)
	}

	rs := NewRunState("test-pipe")
	rs.Steps["build"] = StepState{Status: "done", ExitCode: 0, Output: "ok"}

	if err := Save(rs); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load("test-pipe", rs.RunID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.RunID != rs.RunID {
		t.Fatalf("RunID mismatch: %q vs %q", loaded.RunID, rs.RunID)
	}
	if loaded.PipelineName != "test-pipe" {
		t.Fatalf("PipelineName mismatch: %q", loaded.PipelineName)
	}
	if loaded.Status != "running" {
		t.Fatalf("Status mismatch: %q", loaded.Status)
	}
	ss, ok := loaded.Steps["build"]
	if !ok {
		t.Fatal("missing step 'build'")
	}
	if ss.Status != "done" || ss.Output != "ok" {
		t.Fatalf("step build: status=%q output=%q", ss.Status, ss.Output)
	}
}

func TestSaveLoad_WithSubSteps(t *testing.T) {
	tmp := overrideStateDir(t)
	if err := os.MkdirAll(filepath.Join(tmp, "sub-pipe"), 0o755); err != nil {
		t.Fatal(err)
	}

	rs := NewRunState("sub-pipe")
	rs.Steps["deploy"] = StepState{
		Status: "done",
		SubSteps: map[string]StepState{
			"east": {Status: "done", Output: "east-ok"},
			"west": {Status: "done", Output: "west-ok"},
		},
	}

	if err := Save(rs); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load("sub-pipe", rs.RunID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	subs := loaded.Steps["deploy"].SubSteps
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-steps, got %d", len(subs))
	}
	if subs["east"].Output != "east-ok" {
		t.Fatalf("east output: %q", subs["east"].Output)
	}
	if subs["west"].Output != "west-ok" {
		t.Fatalf("west output: %q", subs["west"].Output)
	}
}

func TestSave_NoTmpFileRemains(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "clean-pipe")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rs := NewRunState("clean-pipe")
	if err := Save(rs); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	entries, err := os.ReadDir(pipeDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("found leftover tmp file: %s", e.Name())
		}
	}
	// Should have exactly one .json file.
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("expected .json file, got %s", entries[0].Name())
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	overrideStateDir(t)
	_, err := Load("nope", "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for missing state file")
	}
	if !strings.Contains(err.Error(), "not found for pipeline") {
		t.Fatalf("expected error containing %q, got %q", "not found for pipeline", err.Error())
	}
}

func TestNewRunState_Defaults(t *testing.T) {
	rs := NewRunState("mypipe")
	if rs.Status != "running" {
		t.Fatalf("expected status %q, got %q", "running", rs.Status)
	}
	if rs.PipelineName != "mypipe" {
		t.Fatalf("expected pipeline name %q, got %q", "mypipe", rs.PipelineName)
	}
	if rs.FinishedAt != nil {
		t.Fatal("expected FinishedAt to be nil")
	}
	if rs.Steps == nil {
		t.Fatal("expected Steps to be initialized")
	}
	if !uuidRe.MatchString(rs.RunID) {
		t.Fatalf("RunID %q is not a valid UUID", rs.RunID)
	}
}
