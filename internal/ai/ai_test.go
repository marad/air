package ai

import (
	"air/internal/util"
	"os"
	"testing"

	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
)

func TestValueOrDefault(t *testing.T) {
	var nilPtr *float32
	setPtr := func(v float32) *float32 { return &v }

	tests := []struct {
		name       string
		ptr        *float32
		defaultVal float32
		want       float32
	}{
		{"nil pointer", nilPtr, 1.0, 1.0},
		{"set pointer", setPtr(2.0), 1.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.ValueOrDefault(tt.ptr, tt.defaultVal)
			if got != tt.want {
				t.Errorf("ValueOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save original env
	original := os.Getenv("TEST_VAR")
	defer os.Setenv("TEST_VAR", original)

	tests := []struct {
		name       string
		envValue   string
		defaultVal string
		want       string
	}{
		{"env set", "value", "default", "value"},
		{"env not set", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_VAR", tt.envValue)
			} else {
				os.Unsetenv("TEST_VAR")
			}
			got := util.GetEnvOrDefault("TEST_VAR", tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelPath(t *testing.T) {
	got := ModelPath("project", "location", "model")
	want := "projects/project/locations/location/publishers/google/models/model"
	if got != want {
		t.Errorf("ModelPath() = %v, want %v", got, want)
	}
}

func TestExtractResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    *aiplatformpb.GenerateContentResponse
		want    *Response
		wantErr bool
	}{
		{
			name: "valid response with metadata",
			resp: &aiplatformpb.GenerateContentResponse{
				Candidates: []*aiplatformpb.Candidate{
					{
						Content: &aiplatformpb.Content{
							Parts: []*aiplatformpb.Part{
								{Data: &aiplatformpb.Part_Text{Text: "Hello, world!"}},
							},
						},
					},
				},
				UsageMetadata: &aiplatformpb.GenerateContentResponse_UsageMetadata{
					PromptTokenCount:     100,
					CandidatesTokenCount: 50,
					TotalTokenCount:      150,
				},
			},
			want: &Response{
				Text:         "Hello, world!",
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
			wantErr: false,
		},
		{
			name: "valid response without metadata",
			resp: &aiplatformpb.GenerateContentResponse{
				Candidates: []*aiplatformpb.Candidate{
					{
						Content: &aiplatformpb.Content{
							Parts: []*aiplatformpb.Part{
								{Data: &aiplatformpb.Part_Text{Text: "Response text"}},
							},
						},
					},
				},
				UsageMetadata: nil,
			},
			want: &Response{
				Text:         "Response text",
				InputTokens:  0,
				OutputTokens: 0,
				TotalTokens:  0,
			},
			wantErr: false,
		},
		{
			name:    "no candidates",
			resp:    &aiplatformpb.GenerateContentResponse{Candidates: []*aiplatformpb.Candidate{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty content",
			resp: &aiplatformpb.GenerateContentResponse{
				Candidates: []*aiplatformpb.Candidate{
					{Content: nil},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty text",
			resp: &aiplatformpb.GenerateContentResponse{
				Candidates: []*aiplatformpb.Candidate{
					{
						Content: &aiplatformpb.Content{
							Parts: []*aiplatformpb.Part{
								{Data: &aiplatformpb.Part_Text{Text: ""}},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractResponse(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Text != tt.want.Text {
					t.Errorf("extractResponse().Text = %v, want %v", got.Text, tt.want.Text)
				}
				if got.InputTokens != tt.want.InputTokens {
					t.Errorf("extractResponse().InputTokens = %v, want %v", got.InputTokens, tt.want.InputTokens)
				}
				if got.OutputTokens != tt.want.OutputTokens {
					t.Errorf("extractResponse().OutputTokens = %v, want %v", got.OutputTokens, tt.want.OutputTokens)
				}
				if got.TotalTokens != tt.want.TotalTokens {
					t.Errorf("extractResponse().TotalTokens = %v, want %v", got.TotalTokens, tt.want.TotalTokens)
				}
			}
		})
	}
}
