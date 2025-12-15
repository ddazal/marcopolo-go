package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParameterProperty represents a single parameter property
type ParameterProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitzero"`
	Default     *string  `json:"default,omitempty"`
}

// Parameter represents the parameters schema for a tool
type Parameters struct {
	Properties map[string]ParameterProperty `json:"properties"`
	Required   []string                     `json:"required"`
}

// Validate checks that all required fields exist in properties
func (p *Parameters) Validate() error {
	for _, requiredKey := range p.Required {
		if _, exists := p.Properties[requiredKey]; !exists {
			return fmt.Errorf("required key %q not found in properties", requiredKey)
		}
	}
	return nil
}

// ToolDefinition represents a tool definition
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  *Parameters `json:"parameters,omitempty"`
}

// Validate validates the tool definition
func (t *ToolDefinition) Validate() error {
	if t.Parameters != nil {
		return t.Parameters.Validate()
	}
	return nil
}

type ToolDescription struct {
	Text        string  `json:"text"`
	InputSchema *string `json:"input_schema,omitempty"`
}

// DescribeTool generates a description for a tool definition
func DescribeTool(toolDef ToolDefinition) (ToolDescription, error) {
	text := strings.Join([]string{
		fmt.Sprintf("Tool: %s", toolDef.Name),
		fmt.Sprintf("Description: %s", toolDef.Description),
	}, "\n")

	result := ToolDescription{
		Text: text,
	}

	if toolDef.Parameters != nil {
		paramsJSON, err := json.Marshal(toolDef.Parameters)
		if err != nil {
			return result, fmt.Errorf("failed to marshal parameters: %w", err)
		}
		paramsStr := string(paramsJSON)
		result.InputSchema = &paramsStr
	}

	return result, nil
}
