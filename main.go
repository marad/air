package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
)

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

	// TODO: Parse frontmatter and content
	_ = content // Placeholder until we implement parsing

	ctx := context.Background()

	result, err := callVertexAI(ctx, "Hello World") // Still hardcoded for now

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calling AI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
