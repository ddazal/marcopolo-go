package tools

import "fmt"

var registry []ToolDefinition

// Register adds a tool definition to the global registry
func Register(tool ToolDefinition) {
	if err := tool.Validate(); err != nil {
		panic(fmt.Sprintf("invalid tool registration: %s - %v", tool.Name, err))
	}
	registry = append(registry, tool)
}

func GetAllTools() []ToolDefinition {
	return registry
}
