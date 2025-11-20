package summary

import (
	"air/internal/ai"
	"fmt"
	"io"
)

// RequestSummary contains information about a completed request
type RequestSummary struct {
	Model        string
	InputTokens  int32
	OutputTokens int32
	TotalTokens  int32
}

// BuildSummary creates a request summary from the model name and AI response
func BuildSummary(model string, response *ai.Response) *RequestSummary {
	return &RequestSummary{
		Model:        model,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		TotalTokens:  response.TotalTokens,
	}
}

// Format returns a formatted string representation of the summary
func (s *RequestSummary) Format() string {
	return fmt.Sprintf(`---
Request Summary
Model: %s
Input tokens: %d
Output tokens: %d
Total tokens: %d
---`,
		s.Model,
		s.InputTokens,
		s.OutputTokens,
		s.TotalTokens,
	)
}

// Display writes the formatted summary to the given writer (typically stderr)
func Display(summary *RequestSummary, writer io.Writer) {
	fmt.Fprintln(writer, summary.Format())
}
