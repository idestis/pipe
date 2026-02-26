package runner

import (
	"testing"

	"github.com/getpipe-dev/pipe/internal/logging"
	"github.com/getpipe-dev/pipe/internal/model"
	"github.com/getpipe-dev/pipe/internal/state"
)

func TestRunInteractive_Echo(t *testing.T) {
	p := &model.Pipeline{
		Name: "test-interactive",
		Steps: []model.Step{
			{ID: "hello", Run: model.RunField{Single: "echo interactive-ok"}, Interactive: true},
		},
	}

	rs := state.NewRunState(p.Name)
	log, err := logging.New(p.Name, rs.RunID)
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	defer func() { _ = log.Close() }()

	r := New(p, rs, log, nil, nil, 0)
	if err := r.Run(); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	ss := rs.Steps["hello"]
	if ss.Status != "done" {
		t.Fatalf("expected status=done, got %q", ss.Status)
	}
	if ss.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", ss.ExitCode)
	}
}

func TestRunInteractive_ResumeSkip(t *testing.T) {
	p := &model.Pipeline{
		Name: "test-interactive-resume",
		Steps: []model.Step{
			{ID: "shell", Run: model.RunField{Single: "false"}, Interactive: true},
		},
	}

	rs := state.NewRunState(p.Name)
	// Pre-populate state as "done" to simulate resume
	rs.Steps["shell"] = state.StepState{Status: "done", ExitCode: 0}

	log, err := logging.New(p.Name, rs.RunID)
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	defer func() { _ = log.Close() }()

	r := New(p, rs, log, nil, nil, 0)
	if err := r.Run(); err != nil {
		t.Fatalf("Run() error: %v (should have skipped the failing command)", err)
	}
}

func TestInteractiveStep_Found(t *testing.T) {
	p := &model.Pipeline{
		Steps: []model.Step{
			{ID: "a", Run: model.RunField{Single: "echo a"}},
			{ID: "b", Run: model.RunField{Single: "bash"}, Interactive: true},
		},
	}
	s := InteractiveStep(p)
	if s == nil {
		t.Fatal("expected to find interactive step")
	}
	if s.ID != "b" {
		t.Fatalf("expected id=b, got %q", s.ID)
	}
}

func TestInteractiveStep_NotFound(t *testing.T) {
	p := &model.Pipeline{
		Steps: []model.Step{
			{ID: "a", Run: model.RunField{Single: "echo a"}},
		},
	}
	if s := InteractiveStep(p); s != nil {
		t.Fatalf("expected nil, got step %q", s.ID)
	}
}
