package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

func overrideCacheDir(t *testing.T) string {
	t.Helper()
	orig := config.CacheDir
	tmp := t.TempDir()
	config.CacheDir = tmp
	t.Cleanup(func() { config.CacheDir = orig })
	return tmp
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	overrideCacheDir(t)

	now := time.Now().Truncate(time.Second)
	exp := now.Add(time.Hour)
	entry := &Entry{
		StepID:    "build",
		CachedAt:  now,
		ExpiresAt: &exp,
		ExitCode:  0,
		Output:    "Build complete\n",
		Sensitive: false,
		RunType:   "single",
	}

	if err := Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("build")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}
	if loaded.StepID != "build" {
		t.Fatalf("expected StepID %q, got %q", "build", loaded.StepID)
	}
	if loaded.Output != "Build complete\n" {
		t.Fatalf("expected Output %q, got %q", "Build complete\n", loaded.Output)
	}
	if loaded.RunType != "single" {
		t.Fatalf("expected RunType %q, got %q", "single", loaded.RunType)
	}
	if loaded.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if !loaded.ExpiresAt.Truncate(time.Second).Equal(exp) {
		t.Fatalf("expected ExpiresAt %v, got %v", exp, *loaded.ExpiresAt)
	}
}

func TestSaveLoad_WithSubOutputs(t *testing.T) {
	overrideCacheDir(t)

	entry := &Entry{
		StepID:   "deploy",
		CachedAt: time.Now(),
		ExitCode: 0,
		RunType:  "subruns",
		SubOutputs: []SubEntry{
			{ID: "prod", Output: "deployed to prod", ExitCode: 0},
			{ID: "staging", Output: "deployed to staging", Sensitive: true, ExitCode: 0},
		},
	}

	if err := Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("deploy")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.SubOutputs) != 2 {
		t.Fatalf("expected 2 sub-outputs, got %d", len(loaded.SubOutputs))
	}
	if loaded.SubOutputs[0].ID != "prod" {
		t.Fatalf("expected sub 0 ID %q, got %q", "prod", loaded.SubOutputs[0].ID)
	}
	if !loaded.SubOutputs[1].Sensitive {
		t.Fatal("expected sub 1 sensitive=true")
	}
}

func TestLoad_Missing(t *testing.T) {
	overrideCacheDir(t)

	entry, err := Load("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Fatal("expected nil entry for missing cache")
	}
}

func TestIsValid_NoExpiry(t *testing.T) {
	entry := &Entry{StepID: "test", CachedAt: time.Now()}
	if !IsValid(entry, time.Now()) {
		t.Fatal("entry with no expiry should always be valid")
	}
}

func TestIsValid_NotExpired(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	entry := &Entry{StepID: "test", ExpiresAt: &exp}
	if !IsValid(entry, time.Now()) {
		t.Fatal("entry should be valid before expiry")
	}
}

func TestIsValid_Expired(t *testing.T) {
	exp := time.Now().Add(-time.Hour)
	entry := &Entry{StepID: "test", ExpiresAt: &exp}
	if IsValid(entry, time.Now()) {
		t.Fatal("entry should be invalid after expiry")
	}
}

func TestIsValid_Nil(t *testing.T) {
	if IsValid(nil, time.Now()) {
		t.Fatal("nil entry should not be valid")
	}
}

func TestClear(t *testing.T) {
	overrideCacheDir(t)

	entry := &Entry{StepID: "test", CachedAt: time.Now(), RunType: "single"}
	if err := Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Clear("test"); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	loaded, err := Load("test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil after Clear")
	}
}

func TestClear_Nonexistent(t *testing.T) {
	overrideCacheDir(t)
	if err := Clear("nonexistent"); err != nil {
		t.Fatalf("Clear nonexistent should not error: %v", err)
	}
}

func TestClearAll(t *testing.T) {
	overrideCacheDir(t)

	for _, id := range []string{"a", "b", "c"} {
		entry := &Entry{StepID: id, CachedAt: time.Now(), RunType: "single"}
		if err := Save(entry); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	if err := ClearAll(); err != nil {
		t.Fatalf("ClearAll: %v", err)
	}

	entries, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after ClearAll, got %d", len(entries))
	}
}

func TestList(t *testing.T) {
	overrideCacheDir(t)

	for _, id := range []string{"x", "y"} {
		entry := &Entry{StepID: id, CachedAt: time.Now(), RunType: "single"}
		if err := Save(entry); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	entries, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestList_EmptyDir(t *testing.T) {
	overrideCacheDir(t)

	entries, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestSave_NoTmpFileRemains(t *testing.T) {
	dir := overrideCacheDir(t)

	entry := &Entry{StepID: "clean", CachedAt: time.Now(), RunType: "single"}
	if err := Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".tmp" {
			t.Fatalf("tmp file leaked: %s", f.Name())
		}
	}
}

func TestSave_SensitiveNoOutput(t *testing.T) {
	overrideCacheDir(t)

	entry := &Entry{
		StepID:    "secret",
		CachedAt:  time.Now(),
		ExitCode:  0,
		Sensitive: true,
		RunType:   "single",
	}

	if err := Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("secret")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Output != "" {
		t.Fatalf("expected empty output for sensitive entry, got %q", loaded.Output)
	}
	if !loaded.Sensitive {
		t.Fatal("expected sensitive=true")
	}
}
