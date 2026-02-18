package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createStateFile creates a fake state file with the given name and sets its
// modification time to baseTime + offset.
func createStateFile(t *testing.T, dir, name string, baseTime time.Time, offsetSec int) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := baseTime.Add(time.Duration(offsetSec) * time.Second)
	if err := os.Chtimes(path, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func TestRotateStates_KeepsNewest(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	currentRunID := "current-run"
	createStateFile(t, pipeDir, currentRunID+".json", base, 100) // newest

	// Create 12 other state files
	for i := range 12 {
		name := "run-" + string(rune('a'+i)) + ".json"
		createStateFile(t, pipeDir, name, base, i)
	}

	if err := RotateStates("demo", currentRunID); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	entries, _ := os.ReadDir(pipeDir)
	// 10 total: 1 current + 9 kept from the 12 candidates
	if len(entries) != 10 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected 10 files, got %d: %v", len(entries), names)
	}

	// Current run must still exist
	if _, err := os.Stat(filepath.Join(pipeDir, currentRunID+".json")); err != nil {
		t.Fatal("current run file was deleted")
	}
}

func TestRotateStates_NeverDeletesCurrentRun(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t.Setenv("PIPE_STATE_ROTATE", "2")

	// Current run is the OLDEST file
	currentRunID := "current-run"
	createStateFile(t, pipeDir, currentRunID+".json", base, 0)

	// Create 5 newer files
	for i := range 5 {
		name := "run-" + string(rune('a'+i)) + ".json"
		createStateFile(t, pipeDir, name, base, i+10)
	}

	if err := RotateStates("demo", currentRunID); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	// Should have 2: current + 1 newest candidate
	entries, _ := os.ReadDir(pipeDir)
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected 2 files, got %d: %v", len(entries), names)
	}

	if _, err := os.Stat(filepath.Join(pipeDir, currentRunID+".json")); err != nil {
		t.Fatal("current run file was deleted")
	}
}

func TestRotateStates_DisabledWithZero(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_STATE_ROTATE", "0")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 15 {
		name := "run-" + string(rune('a'+i)) + ".json"
		createStateFile(t, pipeDir, name, base, i)
	}

	if err := RotateStates("demo", "current-run"); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	entries, _ := os.ReadDir(pipeDir)
	if len(entries) != 15 {
		t.Fatalf("expected 15 files (rotation disabled), got %d", len(entries))
	}
}

func TestRotateStates_CustomEnvValue(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_STATE_ROTATE", "3")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	currentRunID := "current-run"
	createStateFile(t, pipeDir, currentRunID+".json", base, 100)

	for i := range 7 {
		name := "run-" + string(rune('a'+i)) + ".json"
		createStateFile(t, pipeDir, name, base, i)
	}

	if err := RotateStates("demo", currentRunID); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	// 3 total: 1 current + 2 kept
	entries, _ := os.ReadDir(pipeDir)
	if len(entries) != 3 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected 3 files, got %d: %v", len(entries), names)
	}
}

func TestRotateStates_SkipsTmpFiles(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_STATE_ROTATE", "1")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	currentRunID := "current-run"
	createStateFile(t, pipeDir, currentRunID+".json", base, 100)
	createStateFile(t, pipeDir, "old-run.json", base, 0)
	createStateFile(t, pipeDir, "some-run.json.tmp", base, 50)

	if err := RotateStates("demo", currentRunID); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	entries, _ := os.ReadDir(pipeDir)
	// current-run.json (kept, is current) + some-run.json.tmp (skipped, not .json)
	// old-run.json should be deleted
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(entries), names)
	}
}

func TestRotateStates_HubPipeName(t *testing.T) {
	tmp := overrideStateDir(t)
	pipeDir := filepath.Join(tmp, "myowner", "mypipe")
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_STATE_ROTATE", "2")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	currentRunID := "current-run"
	createStateFile(t, pipeDir, currentRunID+".json", base, 100)

	for i := range 5 {
		name := "run-" + string(rune('a'+i)) + ".json"
		createStateFile(t, pipeDir, name, base, i)
	}

	if err := RotateStates("myowner/mypipe", currentRunID); err != nil {
		t.Fatalf("RotateStates error: %v", err)
	}

	entries, _ := os.ReadDir(pipeDir)
	// 2 total: 1 current + 1 kept
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected 2 files, got %d: %v", len(entries), names)
	}
}

func TestRotateStates_EmptyDir(t *testing.T) {
	overrideStateDir(t)
	if err := RotateStates("nonexistent", "some-run"); err != nil {
		t.Fatalf("RotateStates error on missing dir: %v", err)
	}
}
