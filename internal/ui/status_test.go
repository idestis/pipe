package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/getpipe-dev/pipe/internal/model"
)

func steps(ids ...string) []model.Step {
	var out []model.Step
	for _, id := range ids {
		out = append(out, model.Step{
			ID:  id,
			Run: model.RunField{Single: "echo ok"},
		})
	}
	return out
}

func TestNewStatusUI_RowCount(t *testing.T) {
	s := NewStatusUI(&bytes.Buffer{}, steps("a", "b", "c"))
	if len(s.rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(s.rows))
	}
}

func TestNewStatusUI_ParallelStrings(t *testing.T) {
	st := []model.Step{{
		ID:  "lint",
		Run: model.RunField{Strings: []string{"cmd1", "cmd2", "cmd3"}},
	}}
	s := NewStatusUI(&bytes.Buffer{}, st)
	if len(s.rows) != 3 {
		t.Fatalf("expected 3 rows for parallel strings, got %d", len(s.rows))
	}
	if s.rows[0].id != "lint/run_0" {
		t.Fatalf("expected lint/run_0, got %s", s.rows[0].id)
	}
}

func TestNewStatusUI_SubRuns(t *testing.T) {
	st := []model.Step{{
		ID: "deploy",
		Run: model.RunField{SubRuns: []model.SubRun{
			{ID: "api", Run: "deploy api"},
			{ID: "web", Run: "deploy web"},
		}},
	}}
	s := NewStatusUI(&bytes.Buffer{}, st)
	if len(s.rows) != 2 {
		t.Fatalf("expected 2 rows for sub-runs, got %d", len(s.rows))
	}
	if s.rows[1].id != "deploy/web" {
		t.Fatalf("expected deploy/web, got %s", s.rows[1].id)
	}
}

func TestSetStatus_Transitions(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("a"))

	s.SetStatus("a", Running)
	if s.rows[0].status != Running {
		t.Fatal("expected Running")
	}
	if s.rows[0].startedAt.IsZero() {
		t.Fatal("expected startedAt to be set")
	}

	s.SetStatus("a", Done)
	if s.rows[0].status != Done {
		t.Fatal("expected Done")
	}
	if s.rows[0].duration == 0 {
		t.Fatal("expected duration > 0")
	}
}

func TestSetStatus_UnknownID(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("a"))
	// Should not panic
	s.SetStatus("nonexistent", Running)
}

func TestRender_Icons(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))

	s.SetStatus("build", Running)
	out := buf.String()
	if !strings.Contains(out, "●") {
		t.Fatalf("expected ● in output, got: %s", out)
	}
	if !strings.Contains(out, "running...") {
		t.Fatalf("expected 'running...' in output, got: %s", out)
	}

	buf.Reset()
	s.SetStatus("build", Done)
	out = buf.String()
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected ✓ in output, got: %s", out)
	}
}

func TestRender_FailedIcon(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("test"))
	s.SetStatus("test", Running)
	buf.Reset()
	s.SetStatus("test", Failed)
	out := buf.String()
	if !strings.Contains(out, "✗") {
		t.Fatalf("expected ✗ in output, got: %s", out)
	}
}

func TestRender_WaitingIcon(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("push"))
	s.Finish()
	out := buf.String()
	if !strings.Contains(out, "○") {
		t.Fatalf("expected ○ in output, got: %s", out)
	}
	if !strings.Contains(out, "waiting") {
		t.Fatalf("expected 'waiting' in output, got: %s", out)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "(0.5s)"},
		{2100 * time.Millisecond, "(2.1s)"},
		{59900 * time.Millisecond, "(59.9s)"},
		{90 * time.Second, "(1m 30s)"},
		{125 * time.Second, "(2m 5s)"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestPrintAbove_InsertsAboveStatusRows(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build", "deploy"))

	// Initial render to establish status block
	s.SetStatus("build", Running)
	buf.Reset()

	// PrintAbove should move cursor up, print the message, then re-render status
	s.PrintAbove("[build] hello world")
	out := buf.String()

	// Should contain the output line
	if !strings.Contains(out, "[build] hello world") {
		t.Fatalf("expected output line in output, got: %q", out)
	}
	// Should re-render status rows after the output
	if !strings.Contains(out, "build") {
		t.Fatalf("expected status rows re-rendered, got: %q", out)
	}
}

func TestPrintAbove_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))

	s.SetStatus("build", Running)
	buf.Reset()

	s.PrintAbove("[build] line1\n[build] line2")
	out := buf.String()

	if !strings.Contains(out, "[build] line1") {
		t.Fatalf("expected line1 in output, got: %q", out)
	}
	if !strings.Contains(out, "[build] line2") {
		t.Fatalf("expected line2 in output, got: %q", out)
	}
}

func TestAddOutput_FlushedAboveOnDone(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))

	s.SetStatus("build", Running)
	s.AddOutput("build", "compiling main.go")
	s.AddOutput("build", "linking binary")

	buf.Reset()
	s.SetStatus("build", Done)
	out := buf.String()

	// Output should be flushed above with | prefix
	if !strings.Contains(out, "compiling main.go") {
		t.Fatalf("expected 'compiling main.go' flushed above, got: %q", out)
	}
	if !strings.Contains(out, "linking binary") {
		t.Fatalf("expected 'linking binary' flushed above, got: %q", out)
	}
	if !strings.Contains(out, "│") {
		t.Fatalf("expected '|' pipe prefix in output, got: %q", out)
	}

	// After flush, output slice should be cleared
	if len(s.rows[0].output) != 0 {
		t.Fatalf("expected output cleared after flush, got %d lines", len(s.rows[0].output))
	}
}

func TestAddOutput_FlushedAboveOnFailed(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))

	s.SetStatus("build", Running)
	s.AddOutput("build", "error: something broke")
	buf.Reset()
	s.SetStatus("build", Failed)
	out := buf.String()
	if !strings.Contains(out, "error: something broke") {
		t.Fatalf("expected 'error: something broke' flushed above, got: %q", out)
	}
	if !strings.Contains(out, "│") {
		t.Fatalf("expected '|' pipe prefix in output, got: %q", out)
	}
}

func TestAddOutput_NoOutputNoFlush(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))

	s.SetStatus("build", Running)
	buf.Reset()
	s.SetStatus("build", Done)
	out := buf.String()
	// Should not contain pipe character when no output was collected
	if strings.Contains(out, "│") {
		t.Fatalf("expected no pipe output for step without output, got: %q", out)
	}
}

func TestAddOutput_UnknownID(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("build"))
	// Should not panic
	s.AddOutput("nonexistent", "some output")
}

func TestMaxWidth_Alignment(t *testing.T) {
	var buf bytes.Buffer
	s := NewStatusUI(&buf, steps("a", "longname"))
	if s.maxWidth != len("longname") {
		t.Fatalf("expected maxWidth=%d, got %d", len("longname"), s.maxWidth)
	}
}

func TestFlushOutput_PreservesOrder(t *testing.T) {
	var buf bytes.Buffer
	// Pipeline: change-context (no output), pods (with output)
	s := NewStatusUI(&buf, steps("change-context", "pods"))

	// change-context completes first, no output
	s.SetStatus("change-context", Running)
	s.SetStatus("change-context", Done)

	// pods completes second, with output
	s.SetStatus("pods", Running)
	s.AddOutput("pods", "pod/nginx created")
	buf.Reset()
	s.SetStatus("pods", Done)
	out := buf.String()

	// change-context should appear before pods in the flushed output,
	// preserving the original pipeline order.
	ctxIdx := strings.Index(out, "change-context")
	podsIdx := strings.Index(out, "pods")
	if ctxIdx == -1 {
		t.Fatalf("expected 'change-context' in output, got: %q", out)
	}
	if podsIdx == -1 {
		t.Fatalf("expected 'pods' in output, got: %q", out)
	}
	if ctxIdx > podsIdx {
		t.Fatalf("expected change-context before pods, but got change-context at %d, pods at %d\noutput: %q", ctxIdx, podsIdx, out)
	}
}
