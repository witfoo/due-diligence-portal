// Package envconfig provides typed environment variable helpers.
// Ported from the WitFoo Analytics project following the WitFoo Way.
package envconfig

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnv returns the value of the environment variable named by key,
// or defaultValue if the variable is not set or empty.
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvInt returns the integer value of the environment variable named by key,
// or defaultValue if the variable is not set, empty, or not a valid integer.
func GetEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetEnvInt64 returns the int64 value of the environment variable named by key,
// or defaultValue if the variable is not set, empty, or not a valid int64.
func GetEnvInt64(key string, defaultValue int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetEnvBool returns the boolean value of the environment variable named by key,
// or defaultValue if the variable is not set, empty, or not a valid boolean.
func GetEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}

// GetEnvDuration returns the duration value of the environment variable named by key,
// or defaultValue if the variable is not set, empty, or not a valid duration.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

// GetEnvFloat64 returns the float64 value of the environment variable named by key,
// or defaultValue if the variable is not set, empty, or not a valid float64.
func GetEnvFloat64(key string, defaultValue float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// GetEnvList returns the comma-separated list value of the environment variable
// named by key, or defaultValue if the variable is not set or empty.
// Whitespace is trimmed from each entry and empty entries are discarded.
func GetEnvList(key string, defaultValue []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

// MustGetEnv returns the value of the environment variable named by key.
// It panics if the variable is not set or empty.
func MustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}
