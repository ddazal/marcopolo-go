package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ddazal/marcopolo-go/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
