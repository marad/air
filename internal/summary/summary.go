package summary

import (
	"air/internal/ai"
	"fmt"
	"io"
)

type Summary struct {
	Model        string
	InputTokens  int32
	OutputTokens int32
	TotalTokens  int32
}

func BuildSummary(model string, response *ai.Response) *Summary {
	return &Summary{
		Model:        model,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		TotalTokens:  response.TotalTokens,
	}
}

func (s *Summary) Format() string {
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

func Display(summary *Summary, writer io.Writer) {
	fmt.Fprintln(writer, summary.Format())
}
