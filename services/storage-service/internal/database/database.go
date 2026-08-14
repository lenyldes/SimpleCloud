package database

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// BuildDSN constructs a URL-encoded PostgreSQL connection DSN using url.URL.
func BuildDSN(user, password, host, port, dbname, sslmode string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbname,
	}
	q := u.Query()
	if sslmode != "" {
		q.Set("sslmode", sslmode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// InitDB initializes PostgreSQL connection pool and runs embedded migrations.
func InitDB(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	if connString == "" {
		return nil, fmt.Errorf("empty database connection string")
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conn config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return pool, nil
}

// RunMigrations executes embedded SQL scripts against the database with version tracking.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	bootstrapSQL := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ DEFAULT now()
	);`
	if _, err := pool.Exec(ctx, bootstrapSQL); err != nil {
		return fmt.Errorf("failed to execute migration bootstrap: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := entry.Name()

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		var applied bool
		err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if applied {
			_ = tx.Rollback(ctx)
			continue
		}

		sqlContent, err := migrationFS.ReadFile("migrations/" + version)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to read migration file %s: %w", version, err)
		}

		log.Printf("Executing SQL migration: %s", version)
		if _, err := tx.Exec(ctx, string(sqlContent)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}
	}

	return nil
}
