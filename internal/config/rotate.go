package config

import (
	"os"
	"strconv"

	"github.com/charmbracelet/log"
)

// ParseRotateEnv reads an environment variable as an integer rotation limit.
// Unset or empty returns defaultVal. Zero means disabled. Negative or
// non-numeric values return defaultVal with a warning.
func ParseRotateEnv(envName string, defaultVal int) int {
	raw := os.Getenv(envName)
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		log.Warn("invalid rotation limit, using default", "env", envName, "value", raw, "default", defaultVal)
		return defaultVal
	}
	return n
}
