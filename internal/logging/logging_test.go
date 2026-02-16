package logging

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// testLogger returns a Logger that writes to the given buffer (no file).
func testLogger(buf *bytes.Buffer) *Logger {
	return &Logger{w: buf}
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	l.Log("hello %s", "world")

	line := buf.String()
	// Expected: [2026-02-16T10:00:00Z] hello world\n
	re := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] hello world\n$`)
	if !re.MatchString(line) {
		t.Fatalf("unexpected format: %q", line)
	}
}

func TestStepLogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("build", false)
	sl.Log("go build -o app .")

	line := buf.String()
	re := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] \[build\] go build -o app \.\n$`)
	if !re.MatchString(line) {
		t.Fatalf("unexpected format: %q", line)
	}
}

func TestStepLogSensitiveSuppressed(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("secret", true)
	sl.Log("this should not appear")

	if buf.Len() != 0 {
		t.Fatalf("sensitive Log should be no-op, got: %q", buf.String())
	}
}

func TestStepRedacted(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("get-token", true)
	sl.Redacted()

	line := buf.String()
	re := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] \[get-token\] \[SENSITIVE - output redacted\]\n$`)
	if !re.MatchString(line) {
		t.Fatalf("unexpected Redacted format: %q", line)
	}
}

func TestStepExit(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("build", false)
	sl.Exit(0)

	line := buf.String()
	re := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] \[build\] exit 0\n$`)
	if !re.MatchString(line) {
		t.Fatalf("unexpected Exit format: %q", line)
	}
}

func TestStepExitSensitive(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("secret", true)
	sl.Exit(1)

	line := buf.String()
	// Exit is always logged, even for sensitive steps
	if !strings.Contains(line, "[secret] exit 1") {
		t.Fatalf("Exit should always log, got: %q", line)
	}
}

func TestStepWriterNonSensitive(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("build", false)
	w := sl.Writer()

	_, _ = fmt.Fprint(w, "line one\nline two\n")

	out := buf.String()
	if !strings.Contains(out, "[build] line one") {
		t.Fatalf("expected line one in output, got: %q", out)
	}
	if !strings.Contains(out, "[build] line two") {
		t.Fatalf("expected line two in output, got: %q", out)
	}
}

func TestStepWriterSensitiveDiscard(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)
	sl := l.Step("secret", true)
	w := sl.Writer()

	if w != io.Discard {
		t.Fatal("Writer for sensitive step should be io.Discard")
	}
}

func TestConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	l := testLogger(&buf)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sl := l.Step(fmt.Sprintf("step-%d", n), false)
			sl.Log("msg %d", n)
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 100 {
		t.Fatalf("expected 100 lines, got %d", len(lines))
	}

	// Verify no interleaved/corrupted lines
	re := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] \[step-\d+\] msg \d+$`)
	for i, line := range lines {
		if !re.MatchString(line) {
			t.Fatalf("line %d malformed: %q", i, line)
		}
	}
}
