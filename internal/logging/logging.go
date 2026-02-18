package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/idestis/pipe/internal/config"
)

// Logger writes timestamped lines to stderr and a log file.
type Logger struct {
	mu   sync.Mutex
	w    io.Writer
	file *os.File
}

func New(pipelineName, runID string) (*Logger, error) {
	ts := time.Now().Format("20060102-150405")
	rid := runID
	if len(rid) > 8 {
		rid = rid[:8]
	}
	filename := fmt.Sprintf("%s-%s-%s.log", pipelineName, rid, ts)
	path := filepath.Join(config.LogDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}

	return &Logger{
		w:    io.MultiWriter(os.Stderr, f),
		file: f,
	}, nil
}

func (l *Logger) Log(format string, args ...any) {
	ts := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	_, _ = fmt.Fprintf(l.w, "[%s] %s\n", ts, msg)
	l.mu.Unlock()
}

// Step returns a StepLogger scoped to the given step ID.
func (l *Logger) Step(id string, sensitive bool) *StepLogger {
	return &StepLogger{l: l, id: id, sensitive: sensitive}
}

func (l *Logger) Close() error {
	return l.file.Close()
}

// StepLogger writes lines prefixed with the step ID.
type StepLogger struct {
	l         *Logger
	id        string
	sensitive bool
}

// Log writes a timestamped, step-scoped line. No-op if sensitive.
func (s *StepLogger) Log(format string, args ...any) {
	if s.sensitive {
		return
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] %s\n", ts, s.id, msg)
	s.l.mu.Unlock()
}

// Redacted writes a "[SENSITIVE - output redacted]" line.
func (s *StepLogger) Redacted() {
	ts := time.Now().UTC().Format(time.RFC3339)
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] [SENSITIVE - output redacted]\n", ts, s.id)
	s.l.mu.Unlock()
}

// Exit writes an "exit N" line (always logged, even for sensitive steps).
func (s *StepLogger) Exit(code int) {
	ts := time.Now().UTC().Format(time.RFC3339)
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] exit %d\n", ts, s.id, code)
	s.l.mu.Unlock()
}

// Writer returns an io.Writer that routes each line through Log.
// Returns io.Discard for sensitive steps.
func (s *StepLogger) Writer() io.Writer {
	if s.sensitive {
		return io.Discard
	}
	return &stepWriter{sl: s}
}

// stepWriter implements io.Writer, splitting input into lines routed through StepLogger.Log.
type stepWriter struct {
	sl *StepLogger
}

func (w *stepWriter) Write(p []byte) (int, error) {
	s := strings.TrimRight(string(p), "\n")
	if s != "" {
		for _, line := range strings.Split(s, "\n") {
			w.sl.Log("%s", line)
		}
	}
	return len(p), nil
}
