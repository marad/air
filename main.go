package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Generation parameters
	Temperature      *float32 `yaml:"temperature"`      // Pointer to distinguish unset vs 0
	TopP             *float32 `yaml:"topP"`
	MaxTokens        *int32   `yaml:"maxTokens"`
	ResponseMimeType string   `yaml:"responseMimeType"` // "application/json" or "text/plain"
}

func parseFrontmatter(content []byte) (Config, string, error) {
	const prefix = "---\n"
	const delimiter = "\n---\n"

	// Normalize line endings to handle both Unix (\n) and Windows (\r\n) files
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))

	if !bytes.HasPrefix(content, []byte(prefix)) {
		return Config{}, string(content), nil
	}

	// Split content by delimiter to separate frontmatter from markdown
	parts := bytes.SplitN(content, []byte(delimiter), 2)
	if len(parts) < 2 {
		return Config{}, "", fmt.Errorf("invalid frontmatter: missing closing ---")
	}

	var config Config
	if err := yaml.Unmarshal(parts[0][len(prefix):], &config); err != nil {
		return Config{}, "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	markdown := string(parts[1])
	return config, strings.TrimSpace(markdown), nil
}

func loadEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: loading .env: %v\n", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func blockNoSafetySettings() []*aiplatformpb.SafetySetting {
	return []*aiplatformpb.SafetySetting{
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_HATE_SPEECH, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_SEXUALLY_EXPLICIT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_HARASSMENT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
	}
}

func callVertexAI(ctx context.Context, config Config, prompt string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "europe-west1"
	}

	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Apply config with defaults
	temperature := float32(0.0)
	if config.Temperature != nil {
		temperature = *config.Temperature
	}

	topP := float32(0.95)
	if config.TopP != nil {
		topP = *config.TopP
	}

	maxTokens := int32(8192)
	if config.MaxTokens != nil {
		maxTokens = *config.MaxTokens
	}

	responseMimeType := "application/json"
	if config.ResponseMimeType != "" {
		responseMimeType = config.ResponseMimeType
	}

	// Safety settings
	safetySettings := blockNoSafetySettings()

	req := &aiplatformpb.GenerateContentRequest{
		Model: fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/gemini-2.0-flash-001", projectID, location),
		Contents: []*aiplatformpb.Content{
			{
				Role: "user",
				Parts: []*aiplatformpb.Part{
					{Data: &aiplatformpb.Part_Text{Text: prompt}},
				},
			},
		},
		GenerationConfig: &aiplatformpb.GenerationConfig{
			Temperature:      &temperature,
			TopP:             &topP,
			MaxOutputTokens:  &maxTokens,
			ResponseMimeType: responseMimeType,
			//ResponseSchema: &aiplatformpb.Schema{
			//	Type: aiplatformpb.Type_ARRAY,
			//	Items: &aiplatformpb.Schema{
			//		Type: aiplatformpb.Type_OBJECT,
			//		Properties: map[string]*aiplatformpb.Schema{
			//			"check_id": {Type: aiplatformpb.Type_STRING},
			//			"feedback": {Type: aiplatformpb.Type_STRING},
			//			"result":   {Type: aiplatformpb.Type_STRING},
			//		},
			//	},
			//},
		},
		SafetySettings: safetySettings,
	}

	resp, err := client.GenerateContent(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates from model")
	}
	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	// Find the first part with non-empty text
	for _, part := range candidate.Content.Parts {
		if text := part.GetText(); text != "" {
			return text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

func main() {
	loadEnv()

	if len(os.Args) < 2 {
		fatalf("Usage: %s <template_file>", os.Args[0])
	}

	templateFile := os.Args[1]

	content, err := os.ReadFile(templateFile)
	if err != nil {
		fatalf("Error reading file %s: %v", templateFile, err)
	}

	config, markdown, err := parseFrontmatter(content)
	if err != nil {
		fatalf("Error parsing template: %v", err)
	}

	ctx := context.Background()
	result, err := callVertexAI(ctx, config, markdown)
	if err != nil {
		fatalf("Error calling AI: %v", err)
	}

	fmt.Println(result)
}
