package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"slices"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const (
	defaultLocation         = "europe-west1"
	defaultTemperature      = float32(0.0)
	defaultTopP             = float32(0.95)
	defaultMaxTokens        = int32(8192)
	defaultResponseMimeType = "application/json"
	defaultModel            = "gemini-2.0-flash-001"
)

// valueOrDefault returns the dereferenced value if ptr is non-nil, otherwise returns defaultVal.
func valueOrDefault[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func modelPath(projectID, location, model string) string {
	return fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
}

var harmCategoryMap = map[string]aiplatformpb.HarmCategory{
	"hate_speech":       aiplatformpb.HarmCategory_HARM_CATEGORY_HATE_SPEECH,
	"dangerous_content": aiplatformpb.HarmCategory_HARM_CATEGORY_DANGEROUS_CONTENT,
	"sexually_explicit": aiplatformpb.HarmCategory_HARM_CATEGORY_SEXUALLY_EXPLICIT,
	"harassment":        aiplatformpb.HarmCategory_HARM_CATEGORY_HARASSMENT,
}

var safetyThresholdMap = map[string]aiplatformpb.SafetySetting_HarmBlockThreshold{
	"BLOCK_NONE":             aiplatformpb.SafetySetting_BLOCK_NONE,
	"BLOCK_ONLY_HIGH":        aiplatformpb.SafetySetting_BLOCK_ONLY_HIGH,
	"BLOCK_MEDIUM_AND_ABOVE": aiplatformpb.SafetySetting_BLOCK_MEDIUM_AND_ABOVE,
	"BLOCK_LOW_AND_ABOVE":    aiplatformpb.SafetySetting_BLOCK_LOW_AND_ABOVE,
}

// inclusionContext tracks processed files to detect circular includes
type inclusionContext struct {
	visited map[string]bool
	baseDir string
}

func newInclusionContext(initialFile string) *inclusionContext {
	return &inclusionContext{
		visited: make(map[string]bool),
		baseDir: filepath.Dir(initialFile),
	}
}

var includePattern = regexp.MustCompile(`\{\{include\s+"([^"]+)"\}\}`)

func processIncludes(content string, ctx *inclusionContext) (string, error) {
	result := content
	
	for {
		matches := includePattern.FindStringSubmatch(result)
		if matches == nil {
			// No more includes found
			break
		}
		
		includePath := matches[1]
		fullMatch := matches[0]
		
		// Resolve path (relative to current file's directory)
		resolvedPath := includePath
		if !filepath.IsAbs(includePath) {
			resolvedPath = filepath.Join(ctx.baseDir, includePath)
		}
		
		// Normalize path for circular detection
		absPath, err := filepath.Abs(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("resolving include path %s: %w", includePath, err)
		}
		
		// Check for circular includes
		if ctx.visited[absPath] {
			return "", fmt.Errorf("circular include detected: %s", includePath)
		}
		
		// Mark as visited
		ctx.visited[absPath] = true
		
		// Read included file
		includedContent, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("reading included file %s: %w", includePath, err)
		}
		
		// Recursively process includes in the included file
		// Update baseDir for nested includes
		oldBaseDir := ctx.baseDir
		ctx.baseDir = filepath.Dir(absPath)
		
		processedContent, err := processIncludes(string(includedContent), ctx)
		if err != nil {
			return "", err
		}
		
		ctx.baseDir = oldBaseDir
		
		// Replace the include directive with processed content
		result = strings.Replace(result, fullMatch, processedContent, 1)
		
		// Unmark for other branches (allows same file in different paths)
		delete(ctx.visited, absPath)
	}
	
	return result, nil
}

type Config struct {
	Temperature      *float32          `yaml:"temperature"`
	TopP             *float32          `yaml:"topP"`
	MaxTokens        *int32            `yaml:"maxTokens"`
	ResponseMimeType string            `yaml:"responseMimeType"`
	Model            string            `yaml:"model"`
	SafetySettings   map[string]string `yaml:"safetySettings"`
}

func (c *Config) Validate() error {
	if c.Model != "" {
		if err := validateModel(c.Model); err != nil {
			return fmt.Errorf("model: %w", err)
		}
	}

	if len(c.SafetySettings) > 0 {
		if _, err := buildSafetySettings(*c); err != nil {
			return fmt.Errorf("safetySettings: %w", err)
		}
	}

	return nil
}

func parseFrontmatter(content []byte) (Config, string, error) {
	const prefix = "---\n"
	const delimiter = "\n---\n"

	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))

	if !bytes.HasPrefix(content, []byte(prefix)) {
		return Config{}, string(content), nil
	}

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

func validateModel(model string) error {
	supportedModels := []string{
		"gemini-2.0-flash-001",
		"gemini-1.5-pro-002",
		"gemini-1.5-pro-001",
		"gemini-1.5-flash-002",
		"gemini-1.5-flash-001",
	}

	if !slices.Contains(supportedModels, model) {
		return fmt.Errorf("unsupported model: %s (supported: %v)", model, supportedModels)
	}
	return nil
}

func parseHarmCategory(category string) (aiplatformpb.HarmCategory, error) {
	if v, ok := harmCategoryMap[category]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown harm category: %s", category)
}

func parseSafetyThreshold(threshold string) (aiplatformpb.SafetySetting_HarmBlockThreshold, error) {
	if v, ok := safetyThresholdMap[threshold]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown safety threshold: %s", threshold)
}

func buildSafetySettings(config Config) ([]*aiplatformpb.SafetySetting, error) {
	if len(config.SafetySettings) == 0 {
		return blockNoSafetySettings(), nil
	}

	settings := make([]*aiplatformpb.SafetySetting, 0, len(config.SafetySettings))
	for categoryStr, thresholdStr := range config.SafetySettings {
		category, err := parseHarmCategory(categoryStr)
		if err != nil {
			return nil, fmt.Errorf("safety settings: %w", err)
		}

		threshold, err := parseSafetyThreshold(thresholdStr)
		if err != nil {
			return nil, fmt.Errorf("safety settings: %w", err)
		}

		settings = append(settings, &aiplatformpb.SafetySetting{
			Category:  category,
			Threshold: threshold,
		})
	}

	return settings, nil
}

func callVertexAI(ctx context.Context, config Config, prompt string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	location := getEnvOrDefault("GOOGLE_CLOUD_LOCATION", defaultLocation)

	model := defaultModel
	if config.Model != "" {
		if err := validateModel(config.Model); err != nil {
			return "", err
		}
		model = config.Model
	}

	if err := config.Validate(); err != nil {
		return "", err
	}

	client, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	temperature := valueOrDefault(config.Temperature, defaultTemperature)
	topP := valueOrDefault(config.TopP, defaultTopP)
	maxTokens := valueOrDefault(config.MaxTokens, defaultMaxTokens)
	responseMimeType := defaultResponseMimeType
	if config.ResponseMimeType != "" {
		responseMimeType = config.ResponseMimeType
	}

	safetySettings, err := buildSafetySettings(config)
	if err != nil {
		return "", fmt.Errorf("invalid safety settings: %w", err)
	}

	req := &aiplatformpb.GenerateContentRequest{
		Model: modelPath(projectID, location, model),
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
	return text, nil
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

	// Process includes BEFORE parsing frontmatter
	ctx := newInclusionContext(templateFile)
	contentWithIncludes, err := processIncludes(string(content), ctx)
	if err != nil {
		fatalf("Error processing includes: %v", err)
	}

	config, markdown, err := parseFrontmatter([]byte(contentWithIncludes))
	if err != nil {
		fatalf("Error parsing template: %v", err)
	}

	if err := config.Validate(); err != nil {
		fatalf("Invalid configuration: %v", err)
	}

	ctxAI := context.Background()
	result, err := callVertexAI(ctxAI, config, markdown)
	if err != nil {
		fatalf("Error calling AI: %v", err)
	}

	fmt.Println(result)
}
