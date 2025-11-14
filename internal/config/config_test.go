package config

import (
	"testing"

	aiplatform "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantConfig  Config
		wantBody    string
		wantErr     bool
	}{
		{
			name: "valid frontmatter",
			content: `---
temperature: 0.5
model: gemini-1.5-pro-001
---
Hello world`,
			wantConfig: Config{
				Temperature: &[]float32{0.5}[0],
				Model:       "gemini-1.5-pro-001",
			},
			wantBody: "Hello world",
			wantErr:  false,
		},
		{
			name:     "no frontmatter",
			content:  "Hello world",
			wantBody: "Hello world",
			wantErr:  false,
		},
		{
			name:    "invalid YAML",
			content: "---\ninvalid: yaml: content\n---\nHello",
			wantErr: true,
		},
		{
			name:    "missing closing delimiter",
			content: "---\ntemperature: 0.5\nHello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, body, err := ParseFrontmatter([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.Model != tt.wantConfig.Model {
					t.Errorf("ParseFrontmatter() config.Model = %v, want %v", config.Model, tt.wantConfig.Model)
				}
				if tt.wantConfig.Temperature != nil && config.Temperature != nil && *config.Temperature != *tt.wantConfig.Temperature {
					t.Errorf("ParseFrontmatter() config.Temperature = %v, want %v", *config.Temperature, *tt.wantConfig.Temperature)
				}
				if body != tt.wantBody {
					t.Errorf("ParseFrontmatter() body = %v, want %v", body, tt.wantBody)
				}
			}
		})
	}
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{"valid model", "gemini-2.0-flash-001", false},
		{"valid model pro", "gemini-1.5-pro-002", false},
		{"invalid model", "invalid-model", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModel(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseHarmCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		want     aiplatform.HarmCategory
		wantErr  bool
	}{
		{"hate_speech", "hate_speech", aiplatform.HarmCategory_HARM_CATEGORY_HATE_SPEECH, false},
		{"dangerous_content", "dangerous_content", aiplatform.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHarmCategory(tt.category)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHarmCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseHarmCategory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSafetyThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
		want      aiplatform.SafetySetting_HarmBlockThreshold
		wantErr   bool
	}{
		{"BLOCK_NONE", "BLOCK_NONE", aiplatform.SafetySetting_BLOCK_NONE, false},
		{"BLOCK_ONLY_HIGH", "BLOCK_ONLY_HIGH", aiplatform.SafetySetting_BLOCK_ONLY_HIGH, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSafetyThreshold(tt.threshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSafetyThreshold() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSafetyThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSafetySettings(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty safety settings",
			config:  Config{SafetySettings: map[string]string{}},
			wantLen: 4,
			wantErr: false,
		},
		{
			name: "valid safety settings",
			config: Config{SafetySettings: map[string]string{
				"hate_speech": "BLOCK_NONE",
			}},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "invalid category",
			config: Config{SafetySettings: map[string]string{
				"invalid": "BLOCK_NONE",
			}},
			wantErr: true,
		},
		{
			name: "invalid threshold",
			config: Config{SafetySettings: map[string]string{
				"hate_speech": "invalid",
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSafetySettings(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSafetySettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("BuildSafetySettings() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"valid config", Config{Model: "gemini-2.0-flash-001"}, false},
		{"invalid model", Config{Model: "invalid"}, true},
		{"invalid safety category", Config{SafetySettings: map[string]string{"invalid": "BLOCK_NONE"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidateSchema(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"nil schema", Config{ResponseSchema: nil}, false},
		{"valid schema", Config{ResponseSchema: map[string]interface{}{"type": "string"}}, false},
		{"invalid JSON", Config{ResponseSchema: map[string]interface{}{"type": make(chan int)}}, true},
		{"invalid schema", Config{ResponseSchema: map[string]interface{}{"type": "invalid"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateSchema()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}