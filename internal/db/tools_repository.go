package db

import (
	"context"

	"github.com/ddazal/marcopolo-go/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
)

// ToolWithScore wraps a Tool with its relevance score
type ToolWithScore struct {
	*models.Tool
	RelevanceScore float64 `db:"relevance_score"`
}

// ToolRepository defines the interface for tools table database operations.
type ToolRepository interface {
	// UpsertTx inserts or updates a tool within a transaction.
	UpsertTx(ctx context.Context, tx *sqlx.Tx, tool *models.Tool) error

	// FindSimilarWithScore performs vector similarity search with relevance scores.
	// Returns tools with relevance score >= minScore, up to limit results.
	FindSimilarWithScore(ctx context.Context, embedding pgvector.Vector, minScore float64, limit int) ([]*ToolWithScore, error)
}

// PostgresToolRepository implements ToolRepository using PostgreSQL.
type PostgresToolRepository struct {
	db *sqlx.DB
}

// NewPostgresToolRepository creates a new PostgreSQL-backed tool repository.
func NewPostgresToolRepository(db *sqlx.DB) *PostgresToolRepository {
	return &PostgresToolRepository{db: db}
}

// UpsertTx inserts or updates a tool within a transaction.
func (r *PostgresToolRepository) UpsertTx(ctx context.Context, tx *sqlx.Tx, tool *models.Tool) error {
	query := `
		INSERT INTO tools (name, description, embedding, input_schema)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name) WHERE deleted_at IS NULL DO UPDATE SET
			description = EXCLUDED.description,
			embedding = EXCLUDED.embedding,
			input_schema = EXCLUDED.input_schema,
			updated_at = now()
		RETURNING id, created_at, updated_at
	`

	return tx.QueryRowContext(
		ctx,
		query,
		tool.Name,
		tool.Description,
		tool.Embedding,
		tool.InputSchema,
	).Scan(&tool.ID, &tool.CreatedAt, &tool.UpdatedAt)
}

// FindSimilarWithScore performs vector similarity search using cosine distance with scores.
func (r *PostgresToolRepository) FindSimilarWithScore(ctx context.Context, embedding pgvector.Vector, minScore float64, limit int) ([]*ToolWithScore, error) {
	query := `
		SELECT
			id, created_at, updated_at, deleted_at, name, description, embedding, input_schema,
			(1 - (embedding <=> $1))::float8 AS relevance_score
		FROM tools
		WHERE deleted_at IS NULL
		  AND embedding IS NOT NULL
		  AND (1 - (embedding <=> $1)) >= $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	var results []*ToolWithScore
	err := r.db.SelectContext(ctx, &results, query, embedding, minScore, limit)
	if err != nil {
		return nil, err
	}

	return results, nil
}
