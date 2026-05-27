//go:build perf

// Package perf provides shared helpers for performance test suites.
package perf

import (
	"os"
	"strconv"
)

// EnvOrInt reads an integer environment variable, returning defaultVal if unset or invalid.
func EnvOrInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
