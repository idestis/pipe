package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

// ANSI color codes used for verbose-mode terminal output.
const (
	ansiDim   = "\033[2m"
	ansiCyan  = "\033[36m"
	ansiGreen = "\033[32m"
	ansiRed   = "\033[31m"
	ansiReset = "\033[0m"

	// ttyTimeFormat matches the charmbracelet/log format used for debug output.
	ttyTimeFormat = "15:04:05 01/02/2006"
)

// Logger writes timestamped lines to a log file and optionally to the terminal.
type Logger struct {
	mu   sync.Mutex
	w    io.Writer // file writer (always plain text)
	tty  io.Writer // terminal writer (nil in file-only mode)
	file *os.File
}

type option struct{ fileOnly bool }

// Option configures Logger behaviour.
type Option func(*option)

// FileOnly suppresses stderr output; only the log file is written.
func FileOnly() Option { return func(o *option) { o.fileOnly = true } }

func New(pipelineName, runID string, opts ...Option) (*Logger, error) {
	var cfg option
	for _, o := range opts {
		o(&cfg)
	}

	ts := time.Now().Format("20060102-150405")
	rid := runID
	if len(rid) > 8 {
		rid = rid[:8]
	}
	filename := fmt.Sprintf("%s-%s-%s.log", pipelineName, rid, ts)
	path := filepath.Join(config.LogDir, filename)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}

	l := &Logger{
		w:    f,
		file: f,
	}
	if !cfg.fileOnly {
		l.tty = os.Stderr
	}

	return l, nil
}

func (l *Logger) Log(format string, args ...any) {
	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	_, _ = fmt.Fprintf(l.w, "[%s] %s\n", now.UTC().Format(time.RFC3339), msg)
	if l.tty != nil {
		_, _ = fmt.Fprintf(l.tty, "%s[%s]%s %s\n",
			ansiDim, now.Format(ttyTimeFormat), ansiReset, msg)
	}
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
	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] %s\n", now.UTC().Format(time.RFC3339), s.id, msg)
	if s.l.tty != nil {
		_, _ = fmt.Fprintf(s.l.tty, "%s[%s]%s %s[%s]%s %s\n",
			ansiDim, now.Format(ttyTimeFormat), ansiReset, ansiCyan, s.id, ansiReset, msg)
	}
	s.l.mu.Unlock()
}

// Redacted writes a "[SENSITIVE - output redacted]" line.
func (s *StepLogger) Redacted() {
	now := time.Now()
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] [SENSITIVE - output redacted]\n", now.UTC().Format(time.RFC3339), s.id)
	if s.l.tty != nil {
		_, _ = fmt.Fprintf(s.l.tty, "%s[%s]%s %s[%s]%s %s[SENSITIVE - output redacted]%s\n",
			ansiDim, now.Format(ttyTimeFormat), ansiReset, ansiCyan, s.id, ansiReset, ansiDim, ansiReset)
	}
	s.l.mu.Unlock()
}

// Exit writes an "exit N" line (always logged, even for sensitive steps).
func (s *StepLogger) Exit(code int) {
	now := time.Now()
	s.l.mu.Lock()
	_, _ = fmt.Fprintf(s.l.w, "[%s] [%s] exit %d\n", now.UTC().Format(time.RFC3339), s.id, code)
	if s.l.tty != nil {
		exitColor := ansiGreen
		if code != 0 {
			exitColor = ansiRed
		}
		_, _ = fmt.Fprintf(s.l.tty, "%s[%s]%s %s[%s]%s %sexit %d%s\n",
			ansiDim, now.Format(ttyTimeFormat), ansiReset, ansiCyan, s.id, ansiReset, exitColor, code, ansiReset)
	}
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
