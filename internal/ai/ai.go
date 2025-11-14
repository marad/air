package ai

import (
	"context"
	"fmt"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"consistency/internal/config"
	"consistency/internal/schema"
)

func ValueOrDefault[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func GetEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func ModelPath(projectID, location, model string) string {
	return fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
}

func CallVertexAI(ctx context.Context, cfg config.Config, prompt string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	location := GetEnvOrDefault("GOOGLE_CLOUD_LOCATION", config.DefaultLocation)

	model := config.DefaultModel
	if cfg.Model != "" {
		if err := config.ValidateModel(cfg.Model); err != nil {
			return "", err
		}
		model = cfg.Model
	}

	if err := cfg.Validate(); err != nil {
		return "", err
	}

	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	temperature := ValueOrDefault(cfg.Temperature, config.DefaultTemperature)
	topP := ValueOrDefault(cfg.TopP, config.DefaultTopP)
	maxTokens := ValueOrDefault(cfg.MaxTokens, config.DefaultMaxTokens)
	responseMimeType := config.DefaultResponseMimeType
	if cfg.ResponseMimeType != "" {
		responseMimeType = cfg.ResponseMimeType
	}

	safetySettings, err := config.BuildSafetySettings(cfg)
	if err != nil {
		return "", fmt.Errorf("invalid safety settings: %w", err)
	}

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

	// Return the first non-empty text part (prefer first part)
	text := candidate.Content.Parts[0].GetText()
	if text == "" {
		return "", fmt.Errorf("no text content in response")
	}

	// Validate response against schema if provided
	if cfg.ResponseSchema != nil {
		if err := schema.ValidateResponse(text, cfg.ResponseSchema); err != nil {
			fmt.Fprintf(os.Stderr, "warning: response does not match schema: %v\n", err)
		}
	}

	return text, nil
}