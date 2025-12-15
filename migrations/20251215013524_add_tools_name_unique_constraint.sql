-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS tools_name_unique_idx ON tools(name) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS tools_name_unique_idx;
