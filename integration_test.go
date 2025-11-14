package main

import (
	"os"
	"path/filepath"
	"testing"

	"consistency/internal/config"
	"consistency/internal/template"
)

func TestIntegrationConfigAndTemplate(t *testing.T) {
	// Create a test template file
	tempDir, err := os.MkdirTemp(".", "integration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	templateFile := filepath.Join(tempDir, "test.md")
	includedFile := filepath.Join(tempDir, "included.md")

	os.WriteFile(templateFile, []byte(`---
temperature: 0.7
model: gemini-1.5-pro-001
variables:
  name: World
---
Hello {{name}}!

{{include "included.md"}}`), 0644)

	os.WriteFile(includedFile, []byte("This is included content."), 0644)

	// Read the file
	content, err := os.ReadFile(templateFile)
	if err != nil {
		t.Fatal(err)
	}

	// Parse frontmatter
	cfg, body, err := config.ParseFrontmatter(content)
	if err != nil {
		t.Errorf("ParseFrontmatter failed: %v", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}

	// Process includes
	ctx := template.NewInclusionContext(templateFile)
	ctx.BaseDir = tempDir
	processedBody, err := template.ProcessIncludes(body, ctx)
	if err != nil {
		t.Errorf("ProcessIncludes failed: %v", err)
	}

	// Merge variables
	envVars := template.GetEnvVariables()
	cliVars := map[string]string{"cli": "value"}
	allVars := template.MergeVariables(cfg.Variables, envVars, cliVars)

	// Replace placeholders
	finalPrompt, err := template.ReplacePlaceholders(processedBody, allVars)
	if err != nil {
		t.Errorf("ReplacePlaceholders failed: %v", err)
	}

	// Check results
	expected := "Hello World!\n\nThis is included content."
	if finalPrompt != expected {
		t.Errorf("Integration test failed: got %q, want %q", finalPrompt, expected)
	}

	if cfg.Model != "gemini-1.5-pro-001" {
		t.Errorf("Model not parsed correctly: %s", cfg.Model)
	}

	if cfg.Temperature == nil || *cfg.Temperature != 0.7 {
		t.Errorf("Temperature not parsed correctly: %v", cfg.Temperature)
	}
}
