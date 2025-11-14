package schema

import (
	"encoding/json"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func ConvertSchemaToProtobuf(schema map[string]interface{}) *aiplatform.Schema {
	pbSchema := &aiplatform.Schema{}

	typeMap := map[string]aiplatform.Type{
		"string":  aiplatform.Type_STRING,
		"number":  aiplatform.Type_NUMBER,
		"integer": aiplatform.Type_INTEGER,
		"boolean": aiplatform.Type_BOOLEAN,
		"object":  aiplatform.Type_OBJECT,
		"array":   aiplatform.Type_ARRAY,
	}

	if typ, ok := schema["type"].(string); ok {
		if pbType, exists := typeMap[typ]; exists {
			pbSchema.Type = pbType
		}
	}

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		pbSchema.Properties = make(map[string]*aiplatform.Schema)
		for key, val := range properties {
			if propSchema, ok := val.(map[string]interface{}); ok {
				pbSchema.Properties[key] = ConvertSchemaToProtobuf(propSchema)
			}
		}
	}

	if items, ok := schema["items"].(map[string]interface{}); ok {
		pbSchema.Items = ConvertSchemaToProtobuf(items)
	}

	if enum, ok := schema["enum"].([]interface{}); ok {
		pbSchema.Enum = make([]string, len(enum))
		for i, val := range enum {
			if str, ok := val.(string); ok {
				pbSchema.Enum[i] = str
			}
		}
	}

	if required, ok := schema["required"].([]interface{}); ok {
		pbSchema.Required = make([]string, len(required))
		for i, val := range required {
			if str, ok := val.(string); ok {
				pbSchema.Required[i] = str
			}
		}
	}

	return pbSchema
}

func FormatResponse(response string) (string, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(response), &jsonData); err != nil {
		return response, nil // If not JSON, return as is
	}
	formatted, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return response, nil
	}
	return string(formatted), nil
}

func ValidateResponse(response string, schema map[string]interface{}) error {
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	sch, err := jsonschema.CompileString("", string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return sch.Validate(data)
}