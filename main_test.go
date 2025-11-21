package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"air/internal/ai"
	"air/internal/config"
)

func TestRun_MissingArgument(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{} // No template file

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for missing argument")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitInvalidArgs {
		t.Errorf("expected exit code %d, got %d", ExitInvalidArgs, exitErr.code)
	}
}

func TestRun_InvalidFlags(t *testing.T) {
	opts := createTestOptions()
	// Use an actually invalid flag format (missing value)
	opts.args = []string{"--var", "template.md"}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitInvalidArgs {
		t.Errorf("expected exit code %d, got %d", ExitInvalidArgs, exitErr.code)
	}
}

func TestRun_FileNotFound(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"nonexistent.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("file not found")
	}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for file not found")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitFileError {
		t.Errorf("expected exit code %d, got %d", ExitFileError, exitErr.code)
	}
}

func TestRun_InvalidFrontmatter(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("---\ninvalid: yaml: content:\n---\nPrompt text"), nil
	}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for invalid frontmatter")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.code)
	}
}

func TestRun_InvalidConfiguration(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		// Invalid safety threshold
		return []byte("---\nsafetySettings:\n  hate_speech: INVALID_THRESHOLD\n---\nPrompt text"), nil
	}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for invalid configuration")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, exitErr.code)
	}
}

func TestRun_AICallError(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("Simple prompt without frontmatter"), nil
	}
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		return nil, errors.New("API error")
	}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for AI call failure")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitAIError {
		t.Errorf("expected exit code %d, got %d", ExitAIError, exitErr.code)
	}
}

func TestRun_SuccessfulExecution(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	opts := createTestOptions()
	opts.args = []string{"template.md"}
	opts.stdout = stdout
	opts.stderr = stderr
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("---\ntemperature: 0.5\n---\nTest prompt"), nil
	}
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		return &ai.Response{
			Text:        "Test response",
			InputTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	err := run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Test response") {
		t.Errorf("expected output to contain 'Test response', got: %s", output)
	}

	// Check that summary was displayed
	summaryOutput := stderr.String()
	if !strings.Contains(summaryOutput, "Request Summary") {
		t.Errorf("expected summary in stderr, got: %s", summaryOutput)
	}
}

func TestRun_OutputToFile(t *testing.T) {
	writtenFile := ""
	writtenContent := ""

	opts := createTestOptions()
	opts.args = []string{"-o", "output.txt", "template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("Test prompt"), nil
	}
	opts.writeFile = func(path, content string) error {
		writtenFile = path
		writtenContent = content
		return nil
	}
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		return &ai.Response{
			Text:        "File output response",
			InputTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	err := run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writtenFile != "output.txt" {
		t.Errorf("expected file 'output.txt', got: %s", writtenFile)
	}

	if !strings.Contains(writtenContent, "File output response") {
		t.Errorf("expected content to contain 'File output response', got: %s", writtenContent)
	}
}

func TestRun_NoSummary(t *testing.T) {
	stderr := &bytes.Buffer{}

	opts := createTestOptions()
	opts.args = []string{"--no-summary", "template.md"}
	opts.stderr = stderr
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("Test prompt"), nil
	}
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		return &ai.Response{
			Text:        "Response",
			InputTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	err := run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summaryOutput := stderr.String()
	if strings.Contains(summaryOutput, "Input:") {
		t.Errorf("expected no summary with --no-summary flag, got: %s", summaryOutput)
	}
}

func TestRun_WithVariables(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"--var", "name=Alice", "--var", "age=30", "template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("Hello {{name}}, you are {{age}} years old"), nil
	}

	var capturedPrompt string
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		capturedPrompt = prompt
		return &ai.Response{
			Text:        "Response",
			InputTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	err := run(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedPrompt, "Hello Alice") {
		t.Errorf("expected prompt to contain 'Hello Alice', got: %s", capturedPrompt)
	}

	if !strings.Contains(capturedPrompt, "you are 30 years old") {
		t.Errorf("expected prompt to contain 'you are 30 years old', got: %s", capturedPrompt)
	}
}

func TestRun_WriteFileError(t *testing.T) {
	opts := createTestOptions()
	opts.args = []string{"-o", "output.txt", "template.md"}
	opts.readFile = func(path string) ([]byte, error) {
		return []byte("Test prompt"), nil
	}
	opts.writeFile = func(path, content string) error {
		return errors.New("permission denied")
	}
	opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
		return &ai.Response{
			Text:        "Response",
			InputTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	err := run(opts)
	if err == nil {
		t.Fatal("expected error for write file failure")
	}

	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatal("expected exitError")
	}

	if exitErr.code != ExitFileError {
		t.Errorf("expected exit code %d, got %d", ExitFileError, exitErr.code)
	}
}

func TestExitError_Error(t *testing.T) {
	err := &exitError{
		code: ExitAIError,
		err:  errors.New("test error"),
	}

	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got: %s", err.Error())
	}
}

func TestExitError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &exitError{
		code: ExitAIError,
		err:  innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Error("Unwrap() did not return the inner error")
	}
}

func createTestOptions() runOptions {
	return runOptions{
		args:   []string{},
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		readFile: func(path string) ([]byte, error) {
			return []byte("default content"), nil
		},
		writeFile: func(path, content string) error {
			return nil
		},
		getEnvVariables: func() map[string]string {
			return map[string]string{}
		},
		callAI: func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
			return &ai.Response{
				Text:        "default response",
				InputTokens: 10,
				OutputTokens: 20,
			}, nil
		},
	}
}
