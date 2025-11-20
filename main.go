package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"air/internal/ai"
	"air/internal/config"
	"air/internal/schema"
	"air/internal/summary"
	"air/internal/template"
	"github.com/joho/godotenv"
)

const (
	ExitSuccess       = 0
	ExitInvalidArgs   = 2
	ExitFileError     = 3
	ExitConfigError   = 4
	ExitTemplateError = 5
	ExitAIError       = 6
)

// runOptions contains dependencies that can be injected for testing
type runOptions struct {
	args            []string
	stdout          io.Writer
	stderr          io.Writer
	readFile        func(string) ([]byte, error)
	writeFile       func(string, string) error
	getEnvVariables func() map[string]string
	callAI          func(context.Context, config.Config, string) (*ai.Response, error)
}

func loadEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: loading .env: %v\n", err)
	}
}

func fatalf(exitCode int, format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(exitCode)
}

func writeOutputToFile(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}

// run executes the main application logic with injected dependencies
func run(opts runOptions) error {
	// Parse CLI flags
	cliOpts, args, err := template.ParseCLIFlags(opts.args)
	if err != nil {
		return &exitError{code: ExitInvalidArgs, err: fmt.Errorf("parsing flags: %w", err)}
	}

	if len(args) < 1 {
		return &exitError{code: ExitInvalidArgs, err: fmt.Errorf("missing template file argument")}
	}

	templateFile := args[0]

	content, err := opts.readFile(templateFile)
	if err != nil {
		return &exitError{code: ExitFileError, err: fmt.Errorf("reading file %s: %w", templateFile, err)}
	}

	// Process includes BEFORE parsing frontmatter
	includeCtx := template.NewInclusionContext(templateFile)
	contentWithIncludes, err := template.ProcessIncludes(string(content), includeCtx)
	if err != nil {
		return &exitError{code: ExitTemplateError, err: fmt.Errorf("processing includes: %w", err)}
	}

	cfg, markdown, err := config.ParseFrontmatter([]byte(contentWithIncludes))
	if err != nil {
		return &exitError{code: ExitConfigError, err: fmt.Errorf("parsing template: %w", err)}
	}

	if err := cfg.Validate(); err != nil {
		return &exitError{code: ExitConfigError, err: fmt.Errorf("invalid configuration: %w", err)}
	}

	// Merge variables (CLI > frontmatter > env)
	envVars := opts.getEnvVariables()
	variables := template.MergeVariables(envVars, cfg.Variables, cliOpts.Variables)

	// Replace placeholders
	finalMarkdown, err := template.ReplacePlaceholders(markdown, variables)
	if err != nil {
		return &exitError{code: ExitTemplateError, err: fmt.Errorf("replacing placeholders: %w", err)}
	}

	ctx := context.Background()
	response, err := opts.callAI(ctx, cfg, finalMarkdown)
	if err != nil {
		return &exitError{code: ExitAIError, err: fmt.Errorf("calling AI: %w", err)}
	}

	output := response.Text
	if cfg.ResponseSchema != nil {
		output = schema.FormatResponse(response.Text)
	}

	// Write output to file or stdout
	if cliOpts.OutputFile != "" {
		err := opts.writeFile(cliOpts.OutputFile, output)
		if err != nil {
			return &exitError{code: ExitFileError, err: fmt.Errorf("writing output: %w", err)}
		}
	} else {
		fmt.Fprintln(opts.stdout, output)
	}

	// Show summary if not disabled
	if !cliOpts.NoSummary {
		model := cfg.ModelOrDefault()
		s := summary.BuildSummary(model, response)
		summary.Display(s, opts.stderr)
	}

	return nil
}

// exitError wraps an error with an exit code
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

func main() {
	loadEnv()

	opts := runOptions{
		args:            os.Args[1:],
		stdout:          os.Stdout,
		stderr:          os.Stderr,
		readFile:        os.ReadFile,
		writeFile:       writeOutputToFile,
		getEnvVariables: template.GetEnvVariables,
		callAI:          ai.CallVertexAI,
	}

	if err := run(opts); err != nil {
		if exitErr, ok := err.(*exitError); ok {
			fatalf(exitErr.code, "Error: %v", exitErr.err)
		} else {
			fatalf(ExitAIError, "Error: %v", err)
		}
	}
}
