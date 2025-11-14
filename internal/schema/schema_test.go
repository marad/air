package schema

import (
	"testing"

	aiplatform "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
)

func TestConvertSchemaToProtobuf(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]interface{}
		check  func(*aiplatform.Schema) bool
	}{
		{
			name: "string type",
			schema: map[string]interface{}{
				"type": "string",
			},
			check: func(s *aiplatform.Schema) bool {
				return s.Type == aiplatform.Type_STRING
			},
		},
		{
			name: "object with properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			check: func(s *aiplatform.Schema) bool {
				return s.Type == aiplatform.Type_OBJECT && s.Properties["name"].Type == aiplatform.Type_STRING
			},
		},
		{
			name: "array with items",
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{"type": "string"},
			},
			check: func(s *aiplatform.Schema) bool {
				return s.Type == aiplatform.Type_ARRAY && s.Items.Type == aiplatform.Type_STRING
			},
		},
		{
			name: "enum",
			schema: map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"a", "b"},
			},
			check: func(s *aiplatform.Schema) bool {
				return len(s.Enum) == 2 && s.Enum[0] == "a"
			},
		},
		{
			name: "required",
			schema: map[string]interface{}{
				"type":       "object",
				"required":   []interface{}{"name"},
				"properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}},
			},
			check: func(s *aiplatform.Schema) bool {
				return len(s.Required) == 1 && s.Required[0] == "name"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pbSchema := ConvertSchemaToProtobuf(tt.schema)
			if !tt.check(pbSchema) {
				t.Errorf("ConvertSchemaToProtobuf() failed check for %s", tt.name)
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantJSON bool
	}{
		{"JSON response", `{"key": "value"}`, true},
		{"non-JSON response", "plain text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, err := FormatResponse(tt.response)
			if err != nil {
				t.Errorf("FormatResponse() error = %v", err)
				return
			}
			if tt.wantJSON && formatted == tt.response {
				t.Errorf("FormatResponse() should have formatted JSON")
			}
		})
	}
}

func TestValidateResponse(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		schema    map[string]interface{}
		wantErr   bool
	}{
		{
			name:     "valid response",
			response: `{"name": "test"}`,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid JSON",
			response: `invalid json`,
			schema:   map[string]interface{}{"type": "object"},
			wantErr:  true,
		},
		{
			name:     "invalid against schema",
			response: `{"name": 123}`,
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResponse(tt.response, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}