package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"air/internal/ai"
	"air/internal/config"
	"air/internal/schema"
	"air/internal/summary"
	"air/internal/template"
	"github.com/joho/godotenv"
)

const (
	DefaultFileMode = 0644

	ExitSuccess       = 0
	ExitInvalidArgs   = 2
	ExitFileError     = 3
	ExitConfigError   = 4
	ExitTemplateError = 5
	ExitAIError       = 6
)

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
	if strings.Contains(filename, "..") {
		return fmt.Errorf("invalid path: path traversal not allowed")
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
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

func (opts runOptions) writeOutput(cliOpts *template.CLIOptions, content string) error {
	if cliOpts.OutputFile != "" {
		return opts.writeFile(cliOpts.OutputFile, content)
	}
	fmt.Fprintln(opts.stdout, content)
	return nil
}

func run(opts runOptions) error {
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

	envVars := opts.getEnvVariables()
	variables := template.MergeVariables(envVars, cfg.Variables, cliOpts.Variables)

	finalMarkdown, err := template.ReplacePlaceholders(markdown, variables)
	if err != nil {
		return &exitError{code: ExitTemplateError, err: fmt.Errorf("replacing placeholders: %w", err)}
	}

	// If --show-prompt-only flag is set, just output the prompt and exit
	if cliOpts.ShowPromptOnly {
		if err := opts.writeOutput(cliOpts, finalMarkdown); err != nil {
			return &exitError{code: ExitFileError, err: fmt.Errorf("writing output: %w", err)}
		}
		return nil
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

	if err := opts.writeOutput(cliOpts, output); err != nil {
		return &exitError{code: ExitFileError, err: fmt.Errorf("writing output: %w", err)}
	}

	if !cliOpts.NoSummary {
		model := cfg.ModelOrDefault()
		s := summary.BuildSummary(model, response)
		summary.Display(s, opts.stderr)
	}

	return nil
}

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
