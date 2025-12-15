package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ddazal/marcopolo-go/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a PostgreSQL container with pgvector, runs migrations, and returns a DB connection.
func setupTestDB(t *testing.T) *sqlx.DB {
	ctx := context.Background()

	// Start PostgreSQL container with pgvector
	postgresContainer, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	// Cleanup container on test completion
	t.Cleanup(func() {
		require.NoError(t, postgresContainer.Terminate(ctx))
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sqlx.Open("pgx", connStr)
	require.NoError(t, err)

	runMigrations(t, db)

	return db
}

// runMigrations runs all migrations from the migrations directory.
func runMigrations(t *testing.T, db *sqlx.DB) {
	fsys := os.DirFS("../../migrations")
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db.DB,
		fsys,
	)
	require.NoError(t, err)

	_, err = provider.Up(context.Background())
	require.NoError(t, err)
}

func TestPostgresToolRepository_UpsertTx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresToolRepository(db)
	ctx := context.Background()

	tests := map[string]struct {
		setup    func(t *testing.T, tx *sqlx.Tx) (*models.Tool, int64)
		validate func(t *testing.T, expectedID int64, result *models.Tool)
	}{
		"insert new tool": {
			setup: func(_ *testing.T, _ *sqlx.Tx) (*models.Tool, int64) {
				// Create test tool
				embedding := make([]float32, 1536)
				for i := range embedding {
					embedding[i] = float32(i) / 1536.0
				}
				return &models.Tool{
					Name:        "test_tool",
					Description: "A test tool",
					Embedding:   pgvector.NewVector(embedding),
				}, 0
			},
			validate: func(t *testing.T, _ int64, result *models.Tool) {
				// Verify ID and timestamps were set
				assert.NotZero(t, result.ID)
				assert.NotZero(t, result.CreatedAt)
				assert.NotZero(t, result.UpdatedAt)
			},
		},
		"update existing tool": {
			setup: func(t *testing.T, tx *sqlx.Tx) (*models.Tool, int64) {
				embedding1 := make([]float32, 1536)
				for i := range embedding1 {
					embedding1[i] = 0.1
				}

				tool := &models.Tool{
					Name:        "update_test",
					Description: "Original description",
					Embedding:   pgvector.NewVector(embedding1),
				}

				err := repo.UpsertTx(ctx, tx, tool)
				require.NoError(t, err)
				originalID := tool.ID

				// Return tool to update with same name but different data
				embedding2 := make([]float32, 1536)
				for i := range embedding2 {
					embedding2[i] = 0.2
				}

				return &models.Tool{
					Name:        "update_test", // Same name
					Description: "Updated description",
					Embedding:   pgvector.NewVector(embedding2),
				}, originalID // Expect same ID after upsert
			},
			validate: func(t *testing.T, expectedID int64, result *models.Tool) {
				// Should have same ID (update, not insert)
				assert.Equal(t, expectedID, result.ID)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)
			defer tx.Rollback()

			tool, expectedID := tc.setup(t, tx)

			err = repo.UpsertTx(ctx, tx, tool)
			require.NoError(t, err)

			tc.validate(t, expectedID, tool)

			err = tx.Commit()
			require.NoError(t, err)
		})
	}
}

func TestPostgresToolRepository_FindSimilar(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresToolRepository(db)
	ctx := context.Background()

	// Helper to create embedding with all values set to the same float
	makeEmbedding := func(value float32) pgvector.Vector {
		embedding := make([]float32, 1536)
		for i := range embedding {
			embedding[i] = value
		}
		return pgvector.NewVector(embedding)
	}

	tests := map[string]struct {
		seedTools        []*models.Tool
		queryValue       float32
		limit            int
		expectedCount    int
		expectedNames    []string
		notExpectedNames []string
	}{
		"find similar returns closest tools": {
			seedTools: []*models.Tool{
				{Name: "tool1", Description: "First tool", Embedding: makeEmbedding(0.1)},
				{Name: "tool2", Description: "Second tool", Embedding: makeEmbedding(0.5)},
				{Name: "tool3", Description: "Third tool", Embedding: makeEmbedding(0.15)},
			},
			queryValue:       0.12,
			limit:            2,
			expectedCount:    2,
			expectedNames:    []string{"tool1", "tool3"},
			notExpectedNames: []string{"tool2"},
		},
		"respects limit parameter": {
			seedTools: []*models.Tool{
				{Name: "tool1", Description: "First tool", Embedding: makeEmbedding(0.1)},
				{Name: "tool2", Description: "Second tool", Embedding: makeEmbedding(0.5)},
				{Name: "tool3", Description: "Third tool", Embedding: makeEmbedding(0.15)},
			},
			queryValue:    0.2,
			limit:         1,
			expectedCount: 1,
		},
		"returns empty slice when no tools exist": {
			seedTools:     []*models.Tool{},
			queryValue:    0.5,
			limit:         10,
			expectedCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Clean database before each test
			_, err := db.Exec("TRUNCATE tools")
			require.NoError(t, err)

			if len(tc.seedTools) > 0 {
				tx, err := db.Beginx()
				require.NoError(t, err)

				for _, tool := range tc.seedTools {
					require.NoError(t, repo.UpsertTx(ctx, tx, tool))
				}

				require.NoError(t, tx.Commit())
			}

			// Execute query
			query := makeEmbedding(tc.queryValue)
			results, err := repo.FindSimilar(ctx, query, tc.limit)
			require.NoError(t, err)

			assert.Len(t, results, tc.expectedCount)

			if len(tc.expectedNames) > 0 {
				resultNames := make([]string, len(results))
				for i, r := range results {
					resultNames[i] = r.Name
				}

				for _, expectedName := range tc.expectedNames {
					assert.Contains(t, resultNames, expectedName)
				}
			}

			if len(tc.notExpectedNames) > 0 {
				resultNames := make([]string, len(results))
				for i, r := range results {
					resultNames[i] = r.Name
				}

				for _, notExpectedName := range tc.notExpectedNames {
					assert.NotContains(t, resultNames, notExpectedName)
				}
			}
		})
	}
}
