package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"consistency/internal/ai"
	"consistency/internal/config"
	"consistency/internal/schema"
	"consistency/internal/template"
)

const (
	ExitSuccess = 0
	ExitInvalidArgs = 2
	ExitFileError = 3
	ExitConfigError = 4
	ExitTemplateError = 5
	ExitAIError = 6
)

func loadEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: loading .env: %v\n", err)
	}
}

func fatalf(exitCode int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(exitCode)
}

func main() {
	loadEnv()
	
	// Parse CLI flags for variables
	cliVars, args, err := template.ParseVarFlags(os.Args[1:])
	if err != nil {
		fatalf(ExitInvalidArgs, "Error parsing flags: %v", err)
	}
	
	if len(args) < 1 {
		fatalf(ExitInvalidArgs, "Usage: %s [--var key=value ...] <template_file>", os.Args[0])
	}
	
	templateFile := args[0]
	
	content, err := os.ReadFile(templateFile)
	if err != nil {
		fatalf(ExitFileError, "Error reading file %s: %v", templateFile, err)
	}
	
	// Process includes BEFORE parsing frontmatter
	ctx := template.NewInclusionContext(templateFile)
	contentWithIncludes, err := template.ProcessIncludes(string(content), ctx)
	if err != nil {
		fatalf(ExitTemplateError, "Error processing includes: %v", err)
	}
	
	cfg, markdown, err := config.ParseFrontmatter([]byte(contentWithIncludes))
	if err != nil {
		fatalf(ExitConfigError, "Error parsing template: %v", err)
	}
	
	if err := cfg.Validate(); err != nil {
		fatalf(ExitConfigError, "Invalid configuration: %v", err)
	}
	
	// Merge variables (CLI > frontmatter > env)
	envVars := template.GetEnvVariables()
	variables := template.MergeVariables(envVars, cfg.Variables, cliVars)
	
	// Replace placeholders
	finalMarkdown, err := template.ReplacePlaceholders(markdown, variables)
	if err != nil {
		fatalf(ExitTemplateError, "Error replacing placeholders: %v", err)
	}
	
	ctxAI := context.Background()
	result, err := ai.CallVertexAI(ctxAI, cfg, finalMarkdown)
	if err != nil {
		fatalf(ExitAIError, "Error calling AI: %v", err)
	}
	
	if cfg.ResponseSchema != nil {
		formatted, err := schema.FormatResponse(result)
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