package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

//go:embed migrations/catalog.sql migrations/orders.sql
var migrationFiles embed.FS

func Open(ctx context.Context, dataSourceName string) (*sql.DB, error) {
	if strings.TrimSpace(dataSourceName) == "" {
		return nil, fmt.Errorf("database connection string is required")
	}
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL database: %w", err)
	}
	return db, nil
}

func MigrateCatalog(ctx context.Context, db *sql.DB) error {
	return migrate(ctx, db, "catalog", "migrations/catalog.sql", 71243801)
}

func MigrateOrders(ctx context.Context, db *sql.DB) error {
	return migrate(ctx, db, "orders", "migrations/orders.sql", 71243802)
}

func migrate(ctx context.Context, db *sql.DB, service, file string, lockID int64) error {
	content, err := migrationFiles.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s migration: %w", service, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s migration: %w", service, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lockID); err != nil {
		return fmt.Errorf("lock %s migration: %w", service, err)
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("initialize %s migration history: %w", service, err)
	}

	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 1)`).Scan(&applied); err != nil {
		return fmt.Errorf("read %s migration history: %w", service, err)
	}
	if applied {
		return tx.Commit()
	}

	for _, statement := range strings.Split(string(content), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply %s migration statement: %w", service, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (1)`); err != nil {
		return fmt.Errorf("record %s migration: %w", service, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s migration: %w", service, err)
	}
	return nil
}
