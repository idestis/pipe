package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

// overrideLogDir points config.LogDir at a temp directory for the test
// and restores the original value when the test finishes.
func overrideLogDir(t *testing.T) string {
	t.Helper()
	orig := config.LogDir
	tmp := t.TempDir()
	config.LogDir = tmp
	t.Cleanup(func() { config.LogDir = orig })
	return tmp
}

// createLogFile creates a fake log file with the given name and sets its
// modification time to baseTime + offset.
func createLogFile(t *testing.T, dir, name string, baseTime time.Time, offsetSec int) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := baseTime.Add(time.Duration(offsetSec) * time.Second)
	if err := os.Chtimes(path, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func TestRotateLogs_KeepsNewest(t *testing.T) {
	tmp := overrideLogDir(t)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create 12 log files for pipeline "demo"
	for i := range 12 {
		name := "demo-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		if i >= 10 {
			name = "demo-abcdef01-20250101-00001" + string(rune('0'+i%10)) + ".log"
		}
		createLogFile(t, tmp, name, base, i)
	}

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	var remaining []string
	for _, e := range entries {
		remaining = append(remaining, e.Name())
	}
	if len(remaining) != 10 {
		t.Fatalf("expected 10 files, got %d: %v", len(remaining), remaining)
	}
}

func TestRotateLogs_DisabledWithZero(t *testing.T) {
	tmp := overrideLogDir(t)
	t.Setenv("PIPE_LOG_ROTATE", "0")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 15 {
		name := "demo-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		if i >= 10 {
			name = "demo-abcdef01-20250101-00001" + string(rune('0'+i%10)) + ".log"
		}
		createLogFile(t, tmp, name, base, i)
	}

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	if len(entries) != 15 {
		t.Fatalf("expected 15 files (rotation disabled), got %d", len(entries))
	}
}

func TestRotateLogs_CustomEnvValue(t *testing.T) {
	tmp := overrideLogDir(t)
	t.Setenv("PIPE_LOG_ROTATE", "3")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 7 {
		name := "demo-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		createLogFile(t, tmp, name, base, i)
	}

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	if len(entries) != 3 {
		t.Fatalf("expected 3 files, got %d", len(entries))
	}
}

func TestRotateLogs_FewerThanLimit(t *testing.T) {
	tmp := overrideLogDir(t)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 3 {
		name := "demo-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		createLogFile(t, tmp, name, base, i)
	}

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	if len(entries) != 3 {
		t.Fatalf("expected 3 files (fewer than limit), got %d", len(entries))
	}
}

func TestRotateLogs_HubPipeName(t *testing.T) {
	tmp := overrideLogDir(t)
	// Hub pipes like "owner/name" store logs in ~/.pipe/logs/owner/
	ownerDir := filepath.Join(tmp, "myowner")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PIPE_LOG_ROTATE", "2")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		name := "mypipe-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		createLogFile(t, ownerDir, name, base, i)
	}

	if err := RotateLogs("myowner/mypipe"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(ownerDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d", len(entries))
	}
}

func TestRotateLogs_DoesNotMatchSimilarNames(t *testing.T) {
	tmp := overrideLogDir(t)
	t.Setenv("PIPE_LOG_ROTATE", "1")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// "demo" log files
	createLogFile(t, tmp, "demo-abcdef01-20250101-000001.log", base, 1)
	createLogFile(t, tmp, "demo-abcdef01-20250101-000002.log", base, 2)
	// "demo-extra" log file — should not be affected
	createLogFile(t, tmp, "demo-extra-abcdef01-20250101-000001.log", base, 1)

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	// 1 demo + 1 demo-extra
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected 2 files, got %d: %v", len(entries), names)
	}
}

func TestRotateLogs_InvalidEnvFallback(t *testing.T) {
	tmp := overrideLogDir(t)
	t.Setenv("PIPE_LOG_ROTATE", "notanumber")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 12 {
		name := "demo-abcdef01-20250101-00000" + string(rune('0'+i%10)) + ".log"
		if i >= 10 {
			name = "demo-abcdef01-20250101-00001" + string(rune('0'+i%10)) + ".log"
		}
		createLogFile(t, tmp, name, base, i)
	}

	if err := RotateLogs("demo"); err != nil {
		t.Fatalf("RotateLogs error: %v", err)
	}

	entries, _ := os.ReadDir(tmp)
	// Should fall back to default 10
	if len(entries) != 10 {
		t.Fatalf("expected 10 files (fallback default), got %d", len(entries))
	}
}

func TestRotateLogs_EmptyDir(t *testing.T) {
	overrideLogDir(t)
	// No log directory exists — should not error
	if err := RotateLogs("nonexistent"); err != nil {
		t.Fatalf("RotateLogs error on missing dir: %v", err)
	}
}
