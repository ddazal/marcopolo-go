package models

import (
	"time"

	"github.com/ddazal/marcopolo-go/internal/tools"
	"github.com/pgvector/pgvector-go"
)

// Tool represents a persisted tool entity in the database.
// This is separate from ToolDefinition which represents the in-memory tool registry.
type Tool struct {
	ID          int64           `json:"id" db:"id"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Embedding   pgvector.Vector `json:"embedding" db:"embedding"`
	InputSchema *string         `json:"input_schema,omitempty" db:"input_schema"` // JSON string
}

// NewTool creates a Tool entity from a ToolDefinition and embedding.
func NewTool(def tools.ToolDefinition, description tools.ToolDescription, embedding pgvector.Vector) *Tool {
	return &Tool{
		Name:        def.Name,
		Description: description.Text,
		Embedding:   embedding,
		InputSchema: description.InputSchema,
	}
}
