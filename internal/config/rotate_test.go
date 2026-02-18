package config

import (
	"testing"
)

func TestParseRotateEnv_Default(t *testing.T) {
	t.Setenv("PIPE_TEST_ROTATE", "")
	if got := ParseRotateEnv("PIPE_TEST_ROTATE", 10); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func TestParseRotateEnv_Unset(t *testing.T) {
	// Env var not set at all
	if got := ParseRotateEnv("PIPE_TEST_ROTATE_UNSET", 5); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}

func TestParseRotateEnv_Zero(t *testing.T) {
	t.Setenv("PIPE_TEST_ROTATE", "0")
	if got := ParseRotateEnv("PIPE_TEST_ROTATE", 10); got != 0 {
		t.Fatalf("expected 0 (disabled), got %d", got)
	}
}

func TestParseRotateEnv_CustomValue(t *testing.T) {
	t.Setenv("PIPE_TEST_ROTATE", "25")
	if got := ParseRotateEnv("PIPE_TEST_ROTATE", 10); got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
}

func TestParseRotateEnv_Negative(t *testing.T) {
	t.Setenv("PIPE_TEST_ROTATE", "-3")
	if got := ParseRotateEnv("PIPE_TEST_ROTATE", 10); got != 10 {
		t.Fatalf("expected default 10 for negative, got %d", got)
	}
}

func TestParseRotateEnv_Invalid(t *testing.T) {
	t.Setenv("PIPE_TEST_ROTATE", "abc")
	if got := ParseRotateEnv("PIPE_TEST_ROTATE", 10); got != 10 {
		t.Fatalf("expected default 10 for invalid, got %d", got)
	}
}
