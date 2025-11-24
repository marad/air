package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

const (
	DefaultLocation         = "europe-west1"
	DefaultTemperature      = float32(0.0)
	DefaultTopP             = float32(0.95)
	DefaultMaxTokens        = int32(8192)
	DefaultResponseMimeType = "application/json"
	DefaultModel            = "gemini-2.0-flash-001"
)

var HarmCategoryMap = map[string]aiplatform.HarmCategory{
	"hate_speech":       aiplatform.HarmCategory_HARM_CATEGORY_HATE_SPEECH,
	"dangerous_content": aiplatform.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT,
	"sexually_explicit": aiplatform.HarmCategory_HARM_CATEGORY_SEXUALLY_EXPLICIT,
	"harassment":        aiplatform.HarmCategory_HARM_CATEGORY_HARASSMENT,
}

var SafetyThresholdMap = map[string]aiplatform.SafetySetting_HarmBlockThreshold{
	"BLOCK_NONE":             aiplatform.SafetySetting_BLOCK_NONE,
	"BLOCK_ONLY_HIGH":        aiplatform.SafetySetting_BLOCK_ONLY_HIGH,
	"BLOCK_MEDIUM_AND_ABOVE": aiplatform.SafetySetting_BLOCK_MEDIUM_AND_ABOVE,
	"BLOCK_LOW_AND_ABOVE":    aiplatform.SafetySetting_BLOCK_LOW_AND_ABOVE,
}

type Config struct {
	Temperature      *float32               `yaml:"temperature"`
	TopP             *float32               `yaml:"topP"`
	MaxTokens        *int32                 `yaml:"maxTokens"`
	ResponseMimeType string                 `yaml:"responseMimeType"`
	Model            string                 `yaml:"model"`
	SafetySettings   map[string]string      `yaml:"safetySettings"`
	Variables        map[string]string      `yaml:"variables"`
	ResponseSchema   map[string]interface{} `yaml:"responseSchema"`
}

func (c *Config) Validate() error {
	// Validate safety settings without building (BuildSafetySettings will be called later)
	for cat, thresh := range c.SafetySettings {
		if _, err := ParseHarmCategory(cat); err != nil {
			return fmt.Errorf("safetySettings: %w", err)
		}
		if _, err := ParseSafetyThreshold(thresh); err != nil {
			return fmt.Errorf("safetySettings: %w", err)
		}
	}

	return nil
}

// Helper methods for parameter defaults
func (c *Config) TemperatureOrDefault() float32 {
	if c.Temperature != nil {
		return *c.Temperature
	}
	return DefaultTemperature
}

func (c *Config) TopPOrDefault() float32 {
	if c.TopP != nil {
		return *c.TopP
	}
	return DefaultTopP
}

func (c *Config) MaxTokensOrDefault() int32 {
	if c.MaxTokens != nil {
		return *c.MaxTokens
	}
	return DefaultMaxTokens
}

func (c *Config) ResponseMimeTypeOrDefault() string {
	if c.ResponseMimeType != "" {
		return c.ResponseMimeType
	}
	return DefaultResponseMimeType
}

func (c *Config) ModelOrDefault() string {
	if c.Model != "" {
		return c.Model
	}
	return DefaultModel
}

func (c *Config) ValidateSchema() error {
	if c.ResponseSchema == nil {
		return nil
	}

	// Basic validation - ensure it's a valid JSON schema structure
	schemaBytes, err := json.Marshal(c.ResponseSchema)
	if err != nil {
		return fmt.Errorf("invalid response schema: %w", err)
	}

	// Use jsonschema library for validation
	_, err = jsonschema.CompileString("", string(schemaBytes))
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	return nil
}

// ParseFrontmatter extracts YAML frontmatter from markdown content delimited by ---.
func ParseFrontmatter(content []byte) (Config, string, error) {
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))

	if !bytes.HasPrefix(content, []byte("---\n")) {
		return Config{}, string(content), nil
	}

	// Remove leading "---\n"
	content = content[4:]

	// Find closing "---"
	yamlContent, markdown, found := bytes.Cut(content, []byte("\n---\n"))
	if !found {
		return Config{}, "", fmt.Errorf("invalid frontmatter: missing closing ---")
	}

	var config Config
	if len(yamlContent) > 0 {
		if err := yaml.Unmarshal(yamlContent, &config); err != nil {
			return Config{}, "", fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

	return config, strings.TrimSpace(string(markdown)), nil
}

// ParseHarmCategory converts a string harm category to the protobuf enum value.
func ParseHarmCategory(category string) (aiplatform.HarmCategory, error) {
	if v, ok := HarmCategoryMap[category]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown harm category: %s", category)
}

func ParseSafetyThreshold(threshold string) (aiplatform.SafetySetting_HarmBlockThreshold, error) {
	if v, ok := SafetyThresholdMap[threshold]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown safety threshold: %s", threshold)
}

func BuildSafetySettings(config Config) ([]*aiplatform.SafetySetting, error) {
	if len(config.SafetySettings) == 0 {
		return DefaultSafetySettings(), nil
	}

	settings := make([]*aiplatform.SafetySetting, 0, len(config.SafetySettings))
	for categoryStr, thresholdStr := range config.SafetySettings {
		category, err := ParseHarmCategory(categoryStr)
		if err != nil {
			return nil, fmt.Errorf("safety settings: %w", err)
		}

		threshold, err := ParseSafetyThreshold(thresholdStr)
		if err != nil {
			return nil, fmt.Errorf("safety settings: %w", err)
		}

		settings = append(settings, &aiplatform.SafetySetting{
			Category:  category,
			Threshold: threshold,
		})
	}

	return settings, nil
}

func DefaultSafetySettings() []*aiplatform.SafetySetting {
	return []*aiplatform.SafetySetting{
		{Category: aiplatform.HarmCategory_HARM_CATEGORY_HATE_SPEECH, Threshold: aiplatform.SafetySetting_BLOCK_NONE},
		{Category: aiplatform.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT, Threshold: aiplatform.SafetySetting_BLOCK_NONE},
		{Category: aiplatform.HarmCategory_HARM_CATEGORY_SEXUALLY_EXPLICIT, Threshold: aiplatform.SafetySetting_BLOCK_NONE},
		{Category: aiplatform.HarmCategory_HARM_CATEGORY_HARASSMENT, Threshold: aiplatform.SafetySetting_BLOCK_NONE},
	}
}
