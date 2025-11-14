package ai

import (
	"os"
	"testing"
)

func TestValueOrDefault(t *testing.T) {
	var nilPtr *float32
	setPtr := func(v float32) *float32 { return &v }

	tests := []struct {
		name       string
		ptr        *float32
		defaultVal float32
		want       float32
	}{
		{"nil pointer", nilPtr, 1.0, 1.0},
		{"set pointer", setPtr(2.0), 1.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueOrDefault(tt.ptr, tt.defaultVal)
			if got != tt.want {
				t.Errorf("ValueOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save original env
	original := os.Getenv("TEST_VAR")
	defer os.Setenv("TEST_VAR", original)

	tests := []struct {
		name       string
		envValue   string
		defaultVal string
		want       string
	}{
		{"env set", "value", "default", "value"},
		{"env not set", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_VAR", tt.envValue)
			} else {
				os.Unsetenv("TEST_VAR")
			}
			got := GetEnvOrDefault("TEST_VAR", tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelPath(t *testing.T) {
	got := ModelPath("project", "location", "model")
	want := "projects/project/locations/location/publishers/google/models/model"
	if got != want {
		t.Errorf("ModelPath() = %v, want %v", got, want)
	}
}