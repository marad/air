package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"slices"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/joho/godotenv"
	"github.com/santhosh-tekuri/jsonschema/v5"
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

// resolveAbsolutePath resolves a path to absolute, relative to baseDir if not already absolute
func resolveAbsolutePath(path, baseDir string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return filepath.Abs(path)
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
	var result strings.Builder
	lastIndex := 0

	for {
		matches := includePattern.FindStringSubmatch(content[lastIndex:])
		if matches == nil {
			result.WriteString(content[lastIndex:])
			break
		}

		// Calculate absolute position
		matchStart := lastIndex + strings.Index(content[lastIndex:], matches[0])
		matchEnd := matchStart + len(matches[0])

		// Write content before match
		result.WriteString(content[lastIndex:matchStart])

		includePath := matches[1]

		// Resolve path (relative to current file's directory)
		absPath, err := resolveAbsolutePath(includePath, ctx.baseDir)
		if err != nil {
			return "", fmt.Errorf("resolving include path %s: %w", includePath, err)
		}

		// Security check: prevent directory traversal outside the project directory
		projectRoot, err := filepath.Abs(".")
		if err != nil {
			return "", fmt.Errorf("getting project root: %w", err)
		}
		if !strings.HasPrefix(absPath, projectRoot) {
			return "", fmt.Errorf("include path %s is outside the project directory", includePath)
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

		// Write processed content
		result.WriteString(processedContent)

		// Unmark for other branches (allows same file in different paths)
		delete(ctx.visited, absPath)

		lastIndex = matchEnd
	}

	return result.String(), nil
}

var placeholderPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*?)(?:\|([^}]*))?\}\}`)

func replacePlaceholders(content string, variables map[string]string) (string, error) {
	var missing []string

	result := placeholderPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := placeholderPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		if value, ok := variables[varName]; ok {
			return value
		}

		if len(submatches) >= 3 && submatches[2] != "" {
			return submatches[2] // Default value
		}

		// No value and no default - track as missing
		missing = append(missing, varName)
		return match
	})

	if len(missing) > 0 {
		return "", fmt.Errorf("undefined variables without defaults: %v", missing)
	}

	return result, nil
}

func parseVarFlags(args []string) (map[string]string, []string, error) {
	vars := make(map[string]string)
	remaining := []string{}
	
	i := 0
	for i < len(args) {
		arg := args[i]
		
		if arg == "--var" || arg == "-v" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("--var requires an argument")
			}
			
			i++
			varDef := args[i]
			
			// Parse "key=value"
			parts := strings.SplitN(varDef, "=", 2)
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf("invalid --var format: %s (expected key=value)", varDef)
			}
			
			vars[parts[0]] = parts[1]
		} else {
			remaining = append(remaining, arg)
		}
		
		i++
	}
	
	return vars, remaining, nil
}

func getEnvVariables() map[string]string {
	vars := make(map[string]string)
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}
	
	return vars
}

func mergeVariables(sources ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, src := range sources {
		for k, v := range src {
			result[k] = v
		}
	}
	return result
}

type Config struct {
	Temperature      *float32                `yaml:"temperature"`
	TopP             *float32                `yaml:"topP"`
	MaxTokens        *int32                  `yaml:"maxTokens"`
	ResponseMimeType string                  `yaml:"responseMimeType"`
	Model            string                  `yaml:"model"`
	SafetySettings   map[string]string       `yaml:"safetySettings"`
	Variables        map[string]string       `yaml:"variables"`
	ResponseSchema   map[string]interface{}  `yaml:"responseSchema"`
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
	if len(parts[0]) >= len(prefix) {
		yamlContent := parts[0][len(prefix):]
		if len(yamlContent) > 0 {
			if err := yaml.Unmarshal(yamlContent, &config); err != nil {
				return Config{}, "", fmt.Errorf("failed to parse YAML: %w", err)
			}
		}
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

func convertSchemaToProtobuf(schema map[string]interface{}) *aiplatformpb.Schema {
	pbSchema := &aiplatformpb.Schema{}

	typeMap := map[string]aiplatformpb.Type{
		"string":  aiplatformpb.Type_STRING,
		"number":  aiplatformpb.Type_NUMBER,
		"integer": aiplatformpb.Type_INTEGER,
		"boolean": aiplatformpb.Type_BOOLEAN,
		"object":  aiplatformpb.Type_OBJECT,
		"array":   aiplatformpb.Type_ARRAY,
	}

	if typ, ok := schema["type"].(string); ok {
		if pbType, exists := typeMap[typ]; exists {
			pbSchema.Type = pbType
		}
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		pbSchema.Properties = make(map[string]*aiplatformpb.Schema)
		for key, val := range properties {
			if propSchema, ok := val.(map[string]interface{}); ok {
				pbSchema.Properties[key] = convertSchemaToProtobuf(propSchema)
			}
		}
	}

	if items, ok := schema["items"].(map[string]interface{}); ok {
		pbSchema.Items = convertSchemaToProtobuf(items)
	}

	if enum, ok := schema["enum"].([]interface{}); ok {
		pbSchema.Enum = make([]string, len(enum))
		for i, val := range enum {
			if str, ok := val.(string); ok {
				pbSchema.Enum[i] = str
			}
		}
	}

	if required, ok := schema["required"].([]interface{}); ok {
		pbSchema.Required = make([]string, len(required))
		for i, val := range required {
			if str, ok := val.(string); ok {
				pbSchema.Required[i] = str
			}
		}
	}

	return pbSchema
}

func formatResponse(response string) (string, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(response), &jsonData); err != nil {
		return response, nil // If not JSON, return as is
	}
	formatted, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return response, nil
	}
	return string(formatted), nil
}

func validateResponse(response string, schema map[string]interface{}) error {
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	sch, err := jsonschema.CompileString("", string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return sch.Validate(data)
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

	if config.ResponseSchema != nil {
		req.GenerationConfig.ResponseSchema = convertSchemaToProtobuf(config.ResponseSchema)
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
	if config.ResponseSchema != nil {
		if err := validateResponse(text, config.ResponseSchema); err != nil {
			fmt.Fprintf(os.Stderr, "warning: response does not match schema: %v\n", err)
		}
	}

	return text, nil
}

func main() {
	loadEnv()
	
	// Parse CLI flags for variables
	cliVars, args, err := parseVarFlags(os.Args[1:])
	if err != nil {
		fatalf("Error parsing flags: %v", err)
	}
	
	if len(args) < 1 {
		fatalf("Usage: %s [--var key=value ...] <template_file>", os.Args[0])
	}
	
	templateFile := args[0]
	
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
	
	// Merge variables (CLI > frontmatter > env)
	envVars := getEnvVariables()
	variables := mergeVariables(envVars, config.Variables, cliVars)
	
	// Replace placeholders
	finalMarkdown, err := replacePlaceholders(markdown, variables)
	if err != nil {
		fatalf("Error replacing placeholders: %v", err)
	}
	
	ctxAI := context.Background()
	result, err := callVertexAI(ctxAI, config, finalMarkdown)
	if err != nil {
		fatalf("Error calling AI: %v", err)
	}
	
	if config.ResponseSchema != nil {
		formatted, err := formatResponse(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to format response: %v\n", err)
			fmt.Println(result)
		} else {
			fmt.Println(formatted)
		}
	} else {
		fmt.Println(result)
	}
}
