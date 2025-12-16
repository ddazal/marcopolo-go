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

func TestPostgresToolRepository_FindSimilarWithScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresToolRepository(db)
	ctx := context.Background()

	// Helper to create varied embedding vectors for testing similarity
	// Different seeds create embeddings with significantly different cosine similarities
	makeEmbedding := func(seed int) pgvector.Vector {
		embedding := make([]float32, 1536)
		for i := range embedding {
			embedding[i] = float32(seed)*0.1 + float32(i%100)*0.01
		}
		return pgvector.NewVector(embedding)
	}

	tests := map[string]struct {
		seedTools        []*models.Tool
		queryEmbedding   pgvector.Vector
		minScore         float64
		limit            int
		expectedCount    int
		expectedNames    []string
		notExpectedNames []string
		validateScores   bool
	}{
		"returns tools with scores above threshold": {
			seedTools: []*models.Tool{
				{Name: "close_match", Description: "Close match", Embedding: makeEmbedding(5)},
				{Name: "far_match", Description: "Far match", Embedding: makeEmbedding(1)},
			},
			queryEmbedding:   makeEmbedding(5),
			minScore:         0.99,
			limit:            10,
			expectedCount:    1,
			expectedNames:    []string{"close_match"},
			notExpectedNames: []string{"far_match"},
			validateScores:   true,
		},
		"filters by minimum score": {
			seedTools: []*models.Tool{
				{Name: "exact_match", Description: "Exact match tool", Embedding: makeEmbedding(10)},
				{Name: "close_match", Description: "Close match tool", Embedding: makeEmbedding(9)},
				{Name: "far_match", Description: "Far match tool", Embedding: makeEmbedding(1)},
			},
			queryEmbedding:   makeEmbedding(10),
			minScore:         0.98,
			limit:            10,
			expectedCount:    2,
			expectedNames:    []string{"exact_match", "close_match"},
			notExpectedNames: []string{"far_match"},
			validateScores:   true,
		},
		"respects limit parameter": {
			seedTools: []*models.Tool{
				{Name: "tool1", Description: "First tool", Embedding: makeEmbedding(5)},
				{Name: "tool2", Description: "Second tool", Embedding: makeEmbedding(6)},
				{Name: "tool3", Description: "Third tool", Embedding: makeEmbedding(7)},
			},
			queryEmbedding: makeEmbedding(5),
			minScore:       0.0,
			limit:          2,
			expectedCount:  2,
		},
		"returns empty slice when no tools match score threshold": {
			seedTools: []*models.Tool{
				{Name: "tool1", Description: "First tool", Embedding: makeEmbedding(1)},
			},
			queryEmbedding: makeEmbedding(9),
			minScore:       0.99,
			limit:          10,
			expectedCount:  0,
		},
		"returns empty slice when no tools exist": {
			seedTools:      []*models.Tool{},
			queryEmbedding: makeEmbedding(5),
			minScore:       0.7,
			limit:          10,
			expectedCount:  0,
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
			results, err := repo.FindSimilarWithScore(ctx, tc.queryEmbedding, tc.minScore, tc.limit)
			require.NoError(t, err)

			assert.Len(t, results, tc.expectedCount)

			// Validate all returned tools meet the minimum score threshold
			if tc.validateScores {
				for _, result := range results {
					assert.GreaterOrEqual(t, result.RelevanceScore, tc.minScore,
						"Tool %s has score %.4f which is below minimum %.4f",
						result.Name, result.RelevanceScore, tc.minScore)
				}
			}

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
