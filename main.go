package main

import (
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
	// Fields will expand in Milestone 2
	// For now, just parse and ignore
}

func parseFrontmatter(content []byte) (Config, string, error) {
	// Split by "---" delimiters
	// First "---" starts frontmatter
	// Second "---" ends frontmatter
	// Everything after is markdown content

	var config Config
	lines := string(content)

	// Check if file starts with "---"
	if !strings.HasPrefix(lines, "---\n") {
		// No frontmatter, entire content is markdown
		return config, lines, nil
	}

	// Find second "---"
	parts := strings.SplitN(lines[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return config, "", fmt.Errorf("invalid frontmatter: missing closing ---")
	}

	// Parse YAML
	err := yaml.Unmarshal([]byte(parts[0]), &config)
	if err != nil {
		return config, "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Return config and markdown content
	return config, strings.TrimSpace(parts[1]), nil
}

func loadEnv() {
	// Try to load .env from current directory
	err := godotenv.Load()
	if err != nil {
		// Don't fail if .env doesn't exist, just log
		// This is graceful - env vars might already be set
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: error loading .env file: %v\n", err)
		}
	}
}

func callVertexAI(ctx context.Context, prompt string) (string, error) {
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

	temperature := float32(0.0)
	topP := float32(0.95)
	maxTokens := int32(8192)

	// Safety settings
	safetySettings := []*aiplatformpb.SafetySetting{
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_HATE_SPEECH, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_SEXUALLY_EXPLICIT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
		{Category: aiplatformpb.HarmCategory_HARM_CATEGORY_HARASSMENT, Threshold: aiplatformpb.SafetySetting_BLOCK_NONE},
	}

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
			ResponseMimeType: "application/json",
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

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Candidates[0].Content.Parts[0].GetText(), nil
}

func main() {
	// Load .env FIRST, before anything else
	loadEnv()

	// Add argument validation
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <template_file>\n", os.Args[0])
		os.Exit(1)
	}

	templateFile := os.Args[1]

	// Read file contents
	content, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", templateFile, err)
		os.Exit(1)
	}

	// Parse frontmatter
	config, markdown, err := parseFrontmatter(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		os.Exit(1)
	}
	_ = config   // Will use in Milestone 2
	_ = markdown // Will use in integration

	ctx := context.Background()

	result, err := callVertexAI(ctx, "Hello World") // Still hardcoded for now

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calling AI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
