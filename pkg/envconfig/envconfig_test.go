package envconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		want         string
	}{
		{"returns env value when set", "TEST_GET_ENV_1", "hello", "default", "hello"},
		{"returns default when unset", "TEST_GET_ENV_UNSET", "", "default", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			assert.Equal(t, tt.want, GetEnv(tt.key, tt.defaultValue))
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue int
		want         int
	}{
		{"returns int value when set", "TEST_INT_1", "42", 0, 42},
		{"returns default when unset", "TEST_INT_UNSET", "", 10, 10},
		{"returns default for invalid", "TEST_INT_BAD", "notanumber", 10, 10},
		{"handles negative", "TEST_INT_NEG", "-5", 0, -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			assert.Equal(t, tt.want, GetEnvInt(tt.key, tt.defaultValue))
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue bool
		want         bool
	}{
		{"true", "TEST_BOOL", "true", false, true},
		{"1", "TEST_BOOL", "1", false, true},
		{"yes", "TEST_BOOL", "yes", false, true},
		{"on", "TEST_BOOL", "ON", false, true},
		{"false", "TEST_BOOL", "false", true, false},
		{"0", "TEST_BOOL", "0", true, false},
		{"no", "TEST_BOOL", "no", true, false},
		{"off", "TEST_BOOL", "off", true, false},
		{"default true", "TEST_BOOL_UNSET", "", true, true},
		{"default false", "TEST_BOOL_UNSET", "", false, false},
		{"invalid returns default", "TEST_BOOL", "maybe", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			assert.Equal(t, tt.want, GetEnvBool(tt.key, tt.defaultValue))
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("returns parsed duration", func(t *testing.T) {
		t.Setenv("TEST_DUR", "5s")
		assert.Equal(t, 5*time.Second, GetEnvDuration("TEST_DUR", time.Minute))
	})
	t.Run("returns default when unset", func(t *testing.T) {
		assert.Equal(t, time.Minute, GetEnvDuration("TEST_DUR_UNSET", time.Minute))
	})
}

func TestGetEnvList(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue []string
		want         []string
	}{
		{"comma separated", "TEST_LIST", "a,b,c", nil, []string{"a", "b", "c"}},
		{"trims whitespace", "TEST_LIST", " a , b , c ", nil, []string{"a", "b", "c"}},
		{"discards empty", "TEST_LIST", "a,,b,", nil, []string{"a", "b"}},
		{"returns default when unset", "TEST_LIST_UNSET", "", []string{"x"}, []string{"x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			assert.Equal(t, tt.want, GetEnvList(tt.key, tt.defaultValue))
		})
	}
}

func TestMustGetEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_MUST", "value")
		assert.Equal(t, "value", MustGetEnv("TEST_MUST"))
	})
	t.Run("panics when unset", func(t *testing.T) {
		require.Panics(t, func() {
			MustGetEnv("TEST_MUST_UNSET_NONEXISTENT")
		})
	})
}
