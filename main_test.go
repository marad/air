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

func TestRun_ShowPromptOnly(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		fileContent    string
		wantOutput     string
		wantInFile     string
		wantFileName   string
	}{
		{
			name:        "to stdout",
			args:        []string{"--show-prompt-only", "template.md"},
			fileContent: "---\ntemperature: 0.5\n---\nTest prompt with {{var|default}}",
			wantOutput:  "Test prompt with default",
		},
		{
			name:         "to output file",
			args:         []string{"--show-prompt-only", "-o", "prompt.txt", "template.md"},
			fileContent:  "Final prompt {{name|Alice}}",
			wantInFile:   "Final prompt Alice",
			wantFileName: "prompt.txt",
		},
		{
			name:        "with no-summary flag",
			args:        []string{"--show-prompt-only", "--no-summary", "template.md"},
			fileContent: "Simple prompt",
			wantOutput:  "Simple prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			writtenFile := ""
			writtenContent := ""

			opts := createTestOptions()
			opts.args = tt.args
			opts.stdout = stdout
			opts.stderr = stderr
			opts.readFile = func(path string) ([]byte, error) {
				return []byte(tt.fileContent), nil
			}
			opts.writeFile = func(path, content string) error {
				writtenFile = path
				writtenContent = content
				return nil
			}

			aiCalled := false
			opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
				aiCalled = true
				return nil, errors.New("should not be called")
			}

			err := run(opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if aiCalled {
				t.Error("AI should not have been called with --show-prompt-only flag")
			}

			// Check stdout output
			if tt.wantOutput != "" {
				output := stdout.String()
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("expected output to contain %q, got: %s", tt.wantOutput, output)
				}
			}

			// Check file output
			if tt.wantFileName != "" {
				if writtenFile != tt.wantFileName {
					t.Errorf("expected file %q, got: %s", tt.wantFileName, writtenFile)
				}
				if !strings.Contains(writtenContent, tt.wantInFile) {
					t.Errorf("expected content to contain %q, got: %s", tt.wantInFile, writtenContent)
				}
			}

			// Check that summary was NOT displayed
			summaryOutput := stderr.String()
			if strings.Contains(summaryOutput, "Request Summary") {
				t.Errorf("expected no summary with --show-prompt-only flag, got: %s", summaryOutput)
			}
		})
	}
}

func TestRun_ShowPromptOnly_ErrorCases(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		fileContent  string
		wantExitCode int
		wantErrMsg   string
	}{
		{
			name:         "missing variable",
			args:         []string{"--show-prompt-only", "template.md"},
			fileContent:  "Hello {{name}}",
			wantExitCode: ExitTemplateError,
			wantErrMsg:   "undefined variables",
		},
		{
			name:         "invalid config",
			args:         []string{"--show-prompt-only", "template.md"},
			fileContent:  "---\nsafetySettings:\n  hate_speech: INVALID_THRESHOLD\n---\nPrompt",
			wantExitCode: ExitConfigError,
			wantErrMsg:   "invalid configuration",
		},
		{
			name:         "write file error",
			args:         []string{"--show-prompt-only", "-o", "output.txt", "template.md"},
			fileContent:  "Simple prompt",
			wantExitCode: ExitFileError,
			wantErrMsg:   "writing output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := createTestOptions()
			opts.args = tt.args
			opts.readFile = func(path string) ([]byte, error) {
				return []byte(tt.fileContent), nil
			}

			// For write file error test, simulate write failure
			if tt.name == "write file error" {
				opts.writeFile = func(path, content string) error {
					return errors.New("permission denied")
				}
			}

			aiCalled := false
			opts.callAI = func(ctx context.Context, cfg config.Config, prompt string) (*ai.Response, error) {
				aiCalled = true
				return nil, errors.New("should not be called")
			}

			err := run(opts)
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if aiCalled {
				t.Error("AI should not have been called")
			}

			exitErr, ok := err.(*exitError)
			if !ok {
				t.Fatalf("expected exitError, got %T", err)
			}

			if exitErr.code != tt.wantExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, exitErr.code)
			}

			if !strings.Contains(exitErr.Error(), tt.wantErrMsg) {
				t.Errorf("expected error message to contain %q, got: %s", tt.wantErrMsg, exitErr.Error())
			}
		})
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
