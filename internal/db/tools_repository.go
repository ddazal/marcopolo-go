package db

import (
	"context"
	"database/sql"

	"github.com/ddazal/marcopolo-go/internal/models"
	"github.com/pgvector/pgvector-go"
)

// ToolRepository defines the interface for tools table database operations.
type ToolRepository interface {
	// InsertTx creates a new tool record within a transaction.
	InsertTx(ctx context.Context, tx *sql.Tx, tool *models.Tool) error

	// UpsertTx inserts or updates a tool within a transaction.
	UpsertTx(ctx context.Context, tx *sql.Tx, tool *models.Tool) error

	// FindByID retrieves a tool by ID.
	FindByID(ctx context.Context, id int64) (*models.Tool, error)

	// FindByName retrieves a tool by name.
	FindByName(ctx context.Context, name string) (*models.Tool, error)

	// FindSimilar performs vector similarity search.
	// Returns up to limit tools ordered by cosine similarity (most similar first).
	FindSimilar(ctx context.Context, embedding pgvector.Vector, limit int) ([]*models.Tool, error)

	// List retrieves all non-deleted tools.
	List(ctx context.Context) ([]*models.Tool, error)
}

// PostgresToolRepository implements ToolRepository using PostgreSQL.
type PostgresToolRepository struct {
	db *sql.DB
}

// NewPostgresToolRepository creates a new PostgreSQL-backed tool repository.
func NewPostgresToolRepository(db *sql.DB) *PostgresToolRepository {
	return &PostgresToolRepository{db: db}
}

// InsertTx creates a new tool record within a transaction.
func (r *PostgresToolRepository) InsertTx(ctx context.Context, tx *sql.Tx, tool *models.Tool) error {
	query := `
		INSERT INTO tools (name, description, embedding, input_schema)
		VALUES ($1, $2, $3, $4)
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

// UpsertTx inserts or updates a tool within a transaction.
func (r *PostgresToolRepository) UpsertTx(ctx context.Context, tx *sql.Tx, tool *models.Tool) error {
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

// FindByID retrieves a tool by ID.
func (r *PostgresToolRepository) FindByID(ctx context.Context, id int64) (*models.Tool, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, name, description, embedding, input_schema
		FROM tools
		WHERE id = $1 AND deleted_at IS NULL
	`

	tool := &models.Tool{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tool.ID,
		&tool.CreatedAt,
		&tool.UpdatedAt,
		&tool.DeletedAt,
		&tool.Name,
		&tool.Description,
		&tool.Embedding,
		&tool.InputSchema,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	return tool, nil
}

// FindByName retrieves a tool by name.
func (r *PostgresToolRepository) FindByName(ctx context.Context, name string) (*models.Tool, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, name, description, embedding, input_schema
		FROM tools
		WHERE name = $1 AND deleted_at IS NULL
	`

	tool := &models.Tool{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tool.ID,
		&tool.CreatedAt,
		&tool.UpdatedAt,
		&tool.DeletedAt,
		&tool.Name,
		&tool.Description,
		&tool.Embedding,
		&tool.InputSchema,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	return tool, nil
}

// FindSimilar performs vector similarity search using cosine distance.
func (r *PostgresToolRepository) FindSimilar(ctx context.Context, embedding pgvector.Vector, limit int) ([]*models.Tool, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, name, description, embedding, input_schema
		FROM tools
		WHERE deleted_at IS NULL AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Tool
	for rows.Next() {
		tool := &models.Tool{}
		if err := rows.Scan(
			&tool.ID,
			&tool.CreatedAt,
			&tool.UpdatedAt,
			&tool.DeletedAt,
			&tool.Name,
			&tool.Description,
			&tool.Embedding,
			&tool.InputSchema,
		); err != nil {
			return nil, err
		}
		results = append(results, tool)
	}

	return results, rows.Err()
}

// List retrieves all non-deleted tools.
func (r *PostgresToolRepository) List(ctx context.Context) ([]*models.Tool, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, name, description, embedding, input_schema
		FROM tools
		WHERE deleted_at IS NULL
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Tool
	for rows.Next() {
		tool := &models.Tool{}
		if err := rows.Scan(
			&tool.ID,
			&tool.CreatedAt,
			&tool.UpdatedAt,
			&tool.DeletedAt,
			&tool.Name,
			&tool.Description,
			&tool.Embedding,
			&tool.InputSchema,
		); err != nil {
			return nil, err
		}
		results = append(results, tool)
	}

	return results, rows.Err()
}
