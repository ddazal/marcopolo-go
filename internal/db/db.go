package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ddazal/marcopolo-go/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect(ctx context.Context, cfg config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", cfg.DBDSN)
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
