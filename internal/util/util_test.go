package util

import (
	"os"
	"testing"
)

func TestValueOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		ptr         *int
		defaultVal  int
		expected    int
	}{
		{
			name:       "nil pointer returns default",
			ptr:        nil,
			defaultVal: 42,
			expected:   42,
		},
		{
			name:       "non-nil pointer returns value",
			ptr:        intPtr(100),
			defaultVal: 42,
			expected:   100,
		},
		{
			name:       "zero value pointer returns zero",
			ptr:        intPtr(0),
			defaultVal: 42,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueOrDefault(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("ValueOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValueOrDefaultString(t *testing.T) {
	tests := []struct {
		name        string
		ptr         *string
		defaultVal  string
		expected    string
	}{
		{
			name:       "nil string pointer returns default",
			ptr:        nil,
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "non-nil string pointer returns value",
			ptr:        stringPtr("hello"),
			defaultVal: "default",
			expected:   "hello",
		},
		{
			name:       "empty string pointer returns empty",
			ptr:        stringPtr(""),
			defaultVal: "default",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueOrDefault(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("ValueOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		envValue   string
		setEnv     bool
		defaultVal string
		expected   string
	}{
		{
			name:       "env variable exists returns value",
			key:        "TEST_VAR_EXISTS",
			envValue:   "test_value",
			setEnv:     true,
			defaultVal: "default",
			expected:   "test_value",
		},
		{
			name:       "env variable not set returns default",
			key:        "TEST_VAR_NOT_SET",
			envValue:   "",
			setEnv:     false,
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "env variable empty string returns default",
			key:        "TEST_VAR_EMPTY",
			envValue:   "",
			setEnv:     true,
			defaultVal: "default",
			expected:   "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvOrDefault(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetEnvOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper functions
func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}
