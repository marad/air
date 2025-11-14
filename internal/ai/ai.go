package ai

import (
	"context"
	"fmt"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"consistency/internal/config"
	"consistency/internal/schema"
	"consistency/internal/util"
)

func ModelPath(projectID, location, model string) string {
	return fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
}

func loadEnvironment() (projectID, location string, err error) {
	projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	location = util.GetEnvOrDefault("GOOGLE_CLOUD_LOCATION", config.DefaultLocation)
	return projectID, location, nil
}

func buildRequest(cfg config.Config, prompt, projectID, location string) (*aiplatformpb.GenerateContentRequest, error) {
	temperature := cfg.TemperatureOrDefault()
	topP := cfg.TopPOrDefault()
	maxTokens := cfg.MaxTokensOrDefault()
	responseMimeType := cfg.ResponseMimeTypeOrDefault()
	model := cfg.ModelOrDefault()

	safetySettings, err := config.BuildSafetySettings(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid safety settings: %w", err)
	}

	// Note: we take addresses of local variables (temperature, topP, maxTokens)
	// to set the protobuf GenerationConfig fields. This is intentional; in Go
	// these locals will escape to the heap so the pointers remain valid.
	req := &aiplatformpb.GenerateContentRequest{
		Model: ModelPath(projectID, location, model),
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
		},
		SafetySettings: safetySettings,
	}

	if cfg.ResponseSchema != nil {
		req.GenerationConfig.ResponseSchema = schema.ConvertSchemaToProtobuf(cfg.ResponseSchema)
	}

	return req, nil
}

func extractText(resp *aiplatformpb.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("empty response content")
	}

	if text := candidate.Content.Parts[0].GetText(); text != "" {
		return text, nil
	}

	return "", fmt.Errorf("no text in response")
}

func CallVertexAI(ctx context.Context, cfg config.Config, prompt string) (string, error) {
	projectID, location, err := loadEnvironment()
	if err != nil {
		return "", err
	}

	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return "", fmt.Errorf("creating AI client: %w", err)
	}
	defer client.Close()

	req, err := buildRequest(cfg, prompt, projectID, location)
	if err != nil {
		return "", err
	}

	resp, err := client.GenerateContent(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generating content: %w", err)
	}

	text, err := extractText(resp)
	if err != nil {
		return "", err
	}

	// Validate response against schema if provided (just warn, don't fail)
	if cfg.ResponseSchema != nil {
		if err := schema.ValidateResponse(text, cfg.ResponseSchema); err != nil {
			fmt.Fprintf(os.Stderr, "warning: response does not match schema: %v\n", err)
		}
	}

	return text, nil
}
