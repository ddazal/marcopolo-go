# marcopolo-go

An MCP (Model Context Protocol) server that provides semantic search and execution of tools using vector embeddings. Tools are indexed in PostgreSQL with pgvector for similarity search.

## Prerequisites

- Go 1.24.1 or higher
- Docker and Docker Compose
- OpenAI API key (for embeddings)

## Getting Started

### 1. Configure API Key

Create a `config.yaml` file in the project root (you can copy from `config.example.yaml`):

```bash
cp config.example.yaml config.yaml
```

Then edit `config.yaml` and replace `your-openai-api-key-here` with your actual OpenAI API key.

Example configuration:

```yaml
db_dsn: postgres://user:password@localhost:5432/marcopolo?sslmode=disable
embedding:
  provider: "openai"
  model: "text-embedding-3-small"
  api_key: "your-openai-api-key"
```

### 2. Start PostgreSQL Database

The database runs in Docker with the pgvector extension:

```bash
docker-compose up -d
```

This starts PostgreSQL on port 5432 with credentials from `docker-compose.yaml`.

### 3. Run Migrations

Apply all database migrations to set up the schema:

```bash
go run . migrate up
```

This creates the necessary tables and enables the pgvector extension.

### 4. Index Tools

Generate and store embeddings for all registered tools:

```bash
go run . index
```

This command:
1. Loads all tools from the registry
2. Generates embeddings using the configured provider
3. Stores tool metadata and embeddings in the database

You can run this command again to update tools after adding new ones.

### 5. Start the MCP Server

Run the server to handle MCP requests:

```bash
go run . serve
```

The server communicates over stdin/stdout and provides two tools:
- `search_tools`: Find tools using semantic similarity
- `execute_tool`: Run a tool with specified parameters

## Commands

### `index`

Generate embeddings for all tools and store them in the database.

```bash
go run . index
```

Use this after registering new tools or updating existing ones.

### `serve`

Start the MCP server.

```bash
go run . serve
```

The server runs until stopped with Ctrl+C.

### `migrate`

Manage database schema migrations.

#### Apply all pending migrations

```bash
go run . migrate up
```

#### Rollback the last migration

```bash
go run . migrate down
```

#### Check migration status

```bash
go run . migrate status
```

#### Show current migration version

```bash
go run . migrate version
```

#### Create a new migration

```bash
go run . migrate create <name>
```

This creates a timestamped SQL file in the `migrations/` directory.

For Go migrations instead of SQL:

```bash
go run . migrate create <name> --type=go
```

#### Custom migrations directory

All migrate commands accept a `--dir` flag:

```bash
go run . migrate up --dir=custom_migrations
```

#### Verbose output

Enable detailed logging:

```bash
go run . migrate up --verbose
```

## Adding a New Tool

Tools are defined in the `internal/tools/` directory. Each tool needs a definition and an execution handler.

### 1. Create a new file

Create a file like `internal/tools/your_tool.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ddazal/marcopolo-go/internal/mcp"
)

// Define input structure
type YourToolInput struct {
	Param1 string `json:"param1"`
	Param2 int    `json:"param2"`
}

func init() {
	name := "your_tool"

	// Define the tool
	tool := ToolDefinition{
		Name:        name,
		Description: "Brief description of what this tool does",
		Parameters: &Parameters{
			Properties: map[string]ParameterProperty{
				"param1": {
					Type:        "string",
					Description: "Description of param1",
				},
				"param2": {
					Type:        "integer",
					Description: "Description of param2",
				},
			},
			Required: []string{"param1", "param2"},
		},
	}

	// Register the tool definition
	Register(tool)

	// Register the execution handler
	mcp.RegisterExecutable(name, executeYourTool)
}

// Execution handler
func executeYourTool(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var input YourToolInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Implement your tool logic here
	result := fmt.Sprintf("Processed %s with %d", input.Param1, input.Param2)

	return result, nil
}
```

### 2. Re-index tools

After creating the tool, regenerate embeddings:

```bash
go run . index
```

### 3. Test the tool

Restart the MCP server and test your tool through an MCP client.

## Database Migrations

The project uses goose for database migrations. Migration files live in the `migrations/` directory.

### Creating migrations

Generate a new migration:

```bash
go run . migrate create add_your_table
```

This creates two files:
- `migrations/TIMESTAMP_add_your_table.sql` (with `-- +goose Up` and `-- +goose Down` sections)

### Writing migrations

Edit the generated SQL file:

```sql
-- +goose Up
CREATE TABLE your_table (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- +goose Down
DROP TABLE your_table;
```

The `Up` section runs when applying the migration. The `Down` section runs when rolling back.

### Applying migrations

Run pending migrations:

```bash
go run . migrate up
```

### Rolling back migrations

Undo the most recent migration:

```bash
go run . migrate down
```

### Checking status

See which migrations have been applied:

```bash
go run . migrate status
```

## MCP Client Integration

Integrate with Claude Code or Cursor to make the tools available through the AI interface.

### 1. Build the binary

```bash
go build -o marcopolo-go .
```

### 2. Update client configuration

Add the server to your MCP settings file. The file location depends on your client:

Create the file if it does not exist. See `mcp-config.example.json` for a complete example configuration.

Add this configuration:

#### Option A: Using config.yaml file

If you have a `config.yaml` file in the same directory as the binary:

```json
{
  "mcpServers": {
    "marcopolo": {
      "type": "stdio",
      "command": "/absolute/path/to/marcopolo-go",
      "args": ["serve"],
      "env": {}
    }
  }
}
```

#### Option B: Using environment variables

You can override or provide configuration via environment variables instead of `config.yaml`:

```json
{
  "mcpServers": {
    "marcopolo": {
      "type": "stdio",
      "command": "/absolute/path/to/marcopolo-go",
      "args": ["serve"],
      "env": {
        "DB_DSN": "postgres://user:password@localhost:5432/marcopolo?sslmode=disable",
        "EMBEDDING_PROVIDER": "openai",
        "EMBEDDING_MODEL": "text-embedding-3-small",
        "EMBEDDING_API_KEY": "your-openai-api-key"
      }
    }
  }
}
```

**Environment variable mapping:**
- `DB_DSN` → `db_dsn` in config.yaml
- `EMBEDDING_PROVIDER` → `embedding.provider`
- `EMBEDDING_MODEL` → `embedding.model`
- `EMBEDDING_API_KEY` → `embedding.api_key`

Environment variables take precedence over values in `config.yaml`.

Replace `/absolute/path/to/marcopolo-go` with the full path to your built binary.

Note: Claude Code requires the `"type": "stdio"` field, while Cursor works with or without it.

### 3. Restart the client

Restart Claude Code or Cursor to load the MCP server configuration.

### Verifying integration

To verify the MCP server is working:

1. Ask the AI to search for tools
2. The client calls `search_tools` with your query
3. Results show tools ranked by semantic similarity
4. You can then execute tools using `execute_tool`

## Testing

The project includes both unit tests and integration tests using testcontainers.

### Running All Tests

Run the complete test suite:

```bash
go test ./...
```

This executes all tests in the project, including integration tests with testcontainers.

### Running Tests for a Specific Package

Test a single package:

```bash
go test ./internal/db
go test ./internal/embeddings
```

### Running a Specific Test

Execute a single test by name:

```bash
go test ./internal/db -run TestPostgresToolRepository_UpsertTx
```

### Verbose Output

See detailed test output:

```bash
go test ./... -v
```

This shows each test as it runs and any log output.

### Integration Tests with Testcontainers

Integration tests use [testcontainers-go](https://golang.testcontainers.org/) to run PostgreSQL in Docker containers. These tests verify the database layer against a real PostgreSQL instance with pgvector.

**How integration tests work:**

1. **Container setup**: Each test spawns a fresh PostgreSQL container with the pgvector extension
2. **Migration execution**: The test automatically runs all migrations from the `migrations/` directory
3. **Test execution**: Tests run against the live database with real SQL queries
4. **Cleanup**: The container is terminated when the test completes

**Example from `internal/db/tools_repository_test.go`:**

```go
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
```

**Requirements for integration tests:**

- Docker must be running on your machine
- The Docker daemon must be accessible
- Sufficient resources to spawn PostgreSQL containers

**Running only integration tests:**

```bash
go test ./internal/db -v
```

These tests validate:
- Tool insertion and updates (upsert operations)
- Vector similarity search with cosine distance
- Score threshold filtering
- Result limiting and ordering

### Unit Tests with Mocks

Unit tests use mocks to avoid external dependencies. The OpenAI provider tests use an HTTP test server to simulate API responses.

**Example from `internal/embeddings/openai_provider_test.go`:**

```go
func mockOpenAIServer(t *testing.T, response interface{}, statusCode int) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(statusCode)
        json.NewEncoder(w).Encode(response)
    }))
}
```

This approach tests:
- Successful embedding generation
- API error handling
- Response parsing
- Type conversions (float64 to float32)

**Running only unit tests:**

```bash
go test ./internal/embeddings -v
```

### Writing Tests

When adding new features, include tests:

1. **Unit tests**: Mock external dependencies (HTTP calls, file I/O)
2. **Integration tests**: Use testcontainers for database operations
3. **Table-driven tests**: Use maps for multiple test cases (see existing tests for examples)

The project uses [testify](https://github.com/stretchr/testify) for assertions:
- `require.NoError(t, err)`: Fail immediately on error
- `assert.Equal(t, expected, actual)`: Compare values
- `assert.Contains(t, slice, element)`: Check slice membership

## Architecture

The system has three main components:

1. **Tool Registry**: Tools register themselves in `init()` functions. The registry collects all tool definitions.

2. **Embeddings**: Tool descriptions are converted to vectors using OpenAI's embedding API. These vectors enable semantic search.

3. **MCP Server**: Handles two operations:
   - Search: Converts queries to vectors and finds similar tools using pgvector
   - Execute: Runs tools with provided parameters using registered handlers

The database stores tool metadata and embeddings. When you query for tools, the system:
1. Converts your query to an embedding
2. Finds tools with similar embeddings using cosine similarity
3. Returns matching tools with relevance scores
