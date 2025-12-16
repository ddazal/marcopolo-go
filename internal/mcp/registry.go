package mcp

import (
	"context"
	"encoding/json"
	"sync"
)

// ToolHandler is the function signature for executing a tool
// Input is raw JSON arguments, output is the result
type ToolHandler func(ctx context.Context, arguments json.RawMessage) (interface{}, error)

// ExecutableRegistry stores tools with their execution handlers
type ExecutableRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ToolHandler
}

var globalRegistry = &ExecutableRegistry{
	handlers: make(map[string]ToolHandler),
}

// RegisterExecutable registers a tool handler by name
// The toolName should match the tool definition name
func RegisterExecutable(toolName string, handler ToolHandler) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.handlers[toolName] = handler
}

// GetExecutableTool retrieves a tool handler by name
func GetExecutableTool(name string) (ToolHandler, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	handler, exists := globalRegistry.handlers[name]
	return handler, exists
}

// GetAllExecutableToolNames returns all registered tool names
func GetAllExecutableToolNames() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make([]string, 0, len(globalRegistry.handlers))
	for name := range globalRegistry.handlers {
		result = append(result, name)
	}
	return result
}
