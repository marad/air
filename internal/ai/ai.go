package ai

import (
	"context"
	"fmt"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"air/internal/config"
	"air/internal/schema"
	"air/internal/util"
)

// Response represents the AI response with metadata
type Response struct {
	Text         string
	InputTokens  int32
	OutputTokens int32
	TotalTokens  int32
}

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

func extractResponse(resp *aiplatformpb.GenerateContentResponse) (*Response, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	text := candidate.Content.Parts[0].GetText()
	if text == "" {
		return nil, fmt.Errorf("no text in response")
	}

	result := &Response{
		Text: text,
	}

	if resp.UsageMetadata != nil {
		result.InputTokens = resp.UsageMetadata.PromptTokenCount
		result.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
		result.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return result, nil
}

func CallVertexAI(ctx context.Context, cfg config.Config, prompt string) (*Response, error) {
	projectID, location, err := loadEnvironment()
	if err != nil {
		return nil, err
	}

	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating AI client: %w", err)
	}
	defer client.Close()

	req, err := buildRequest(cfg, prompt, projectID, location)
	if err != nil {
		return nil, err
	}

	resp, err := client.GenerateContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	response, err := extractResponse(resp)
	if err != nil {
		return nil, err
	}

	// Validate response against schema if provided (just warn, don't fail)
	if cfg.ResponseSchema != nil {
		if err := schema.ValidateResponse(response.Text, cfg.ResponseSchema); err != nil {
			fmt.Fprintf(os.Stderr, "warning: response does not match schema: %v\n", err)
		}
	}

	return response, nil
}
