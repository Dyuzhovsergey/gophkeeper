package config

import (
	"os"
	"strings"
)

const defaultLogLevel = "info"

func applyStringEnv(dst *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*dst = strings.TrimSpace(value)
	}
}

func normalizeLogLevel(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
