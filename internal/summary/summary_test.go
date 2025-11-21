package summary

import (
	"air/internal/ai"
	"bytes"
	"strings"
	"testing"
)

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		response *ai.Response
	}{
		{
			name:  "gemini-2.0-flash-001",
			model: "gemini-2.0-flash-001",
			response: &ai.Response{
				Text:         "Test response",
				InputTokens:  1000,
				OutputTokens: 500,
				TotalTokens:  1500,
			},
		},
		{
			name:  "zero tokens",
			model: "gemini-1.5-pro-002",
			response: &ai.Response{
				Text:         "Test response",
				InputTokens:  0,
				OutputTokens: 0,
				TotalTokens:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := BuildSummary(tt.model, tt.response)
			if summary.Model != tt.model {
				t.Errorf("BuildSummary().Model = %v, want %v", summary.Model, tt.model)
			}
			if summary.InputTokens != tt.response.InputTokens {
				t.Errorf("BuildSummary().InputTokens = %v, want %v", summary.InputTokens, tt.response.InputTokens)
			}
			if summary.OutputTokens != tt.response.OutputTokens {
				t.Errorf("BuildSummary().OutputTokens = %v, want %v", summary.OutputTokens, tt.response.OutputTokens)
			}
			if summary.TotalTokens != tt.response.TotalTokens {
				t.Errorf("BuildSummary().TotalTokens = %v, want %v", summary.TotalTokens, tt.response.TotalTokens)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	summary := &Summary{
		Model:        "gemini-2.0-flash-001",
		InputTokens:  1234,
		OutputTokens: 567,
		TotalTokens:  1801,
	}

	formatted := summary.Format()

	if !strings.Contains(formatted, "Request Summary") {
		t.Error("Format() should contain 'Request Summary'")
	}
	if !strings.Contains(formatted, "gemini-2.0-flash-001") {
		t.Error("Format() should contain model name")
	}
	if !strings.Contains(formatted, "1234") {
		t.Error("Format() should contain input tokens")
	}
	if !strings.Contains(formatted, "567") {
		t.Error("Format() should contain output tokens")
	}
	if !strings.Contains(formatted, "1801") {
		t.Error("Format() should contain total tokens")
	}
	if !strings.Contains(formatted, "---") {
		t.Error("Format() should contain separator lines")
	}
}

func TestDisplay(t *testing.T) {
	summary := &Summary{
		Model:        "gemini-2.0-flash-001",
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	var buf bytes.Buffer
	Display(summary, &buf)

	output := buf.String()
	if output == "" {
		t.Error("Display() should write to writer")
	}
	if !strings.Contains(output, "Request Summary") {
		t.Error("Display() output should contain 'Request Summary'")
	}
	if !strings.Contains(output, "gemini-2.0-flash-001") {
		t.Error("Display() output should contain model name")
	}
}
