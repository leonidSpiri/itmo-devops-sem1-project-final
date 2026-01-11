package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.postgresConnString())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := ensureSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func ensureSchema(ctx context.Context, db *sql.DB) error {
	const q = `
CREATE TABLE IF NOT EXISTS prices (
	id           BIGINT      NOT NULL,
	create_date  DATE        NOT NULL,
	name         TEXT        NOT NULL,
	category     TEXT        NOT NULL,
	price        NUMERIC     NOT NULL
);
`
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("create table prices: %w", err)
	}
	return nil
}
