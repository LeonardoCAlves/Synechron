package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

//go:embed migrations/catalog.sql migrations/catalog_v2.sql migrations/catalog_v3.sql migrations/orders.sql migrations/orders_v2.sql migrations/orders_v3.sql
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
	return migrate(ctx, db, "catalog", []migration{
		{version: 1, file: "migrations/catalog.sql"},
		{version: 2, file: "migrations/catalog_v2.sql"},
		{version: 3, file: "migrations/catalog_v3.sql"},
	}, 71243801)
}

func MigrateOrders(ctx context.Context, db *sql.DB) error {
	return migrate(ctx, db, "orders", []migration{
		{version: 1, file: "migrations/orders.sql"},
		{version: 2, file: "migrations/orders_v2.sql"},
		{version: 3, file: "migrations/orders_v3.sql"},
	}, 71243802)
}

type migration struct {
	version int
	file    string
}

func migrate(ctx context.Context, db *sql.DB, service string, migrations []migration, lockID int64) error {

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

	for _, migration := range migrations {
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("read %s migration history: %w", service, err)
		}
		if applied {
			continue
		}
		content, err := migrationFiles.ReadFile(migration.file)
		if err != nil {
			return fmt.Errorf("read %s migration %d: %w", service, migration.version, err)
		}
		for _, statement := range strings.Split(string(content), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply %s migration %d statement: %w", service, migration.version, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, migration.version); err != nil {
			return fmt.Errorf("record %s migration %d: %w", service, migration.version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s migration: %w", service, err)
	}
	return nil
}
