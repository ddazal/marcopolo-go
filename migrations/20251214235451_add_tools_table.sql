-- +goose Up
CREATE TABLE IF NOT EXISTS tools (
    id bigserial primary key ,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    deleted_at timestamptz,
    name text not null,
    description text not null,
    embedding vector(1536),
    input_schema jsonb
);

-- +goose Down
DROP TABLE IF EXISTS tools;
