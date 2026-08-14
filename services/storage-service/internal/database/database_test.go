package database_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/database"
)

func TestInitDB_ErrorCases(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Run("Empty connection string", func(t *testing.T) {
		_, err := database.InitDB(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty connection string, got nil")
		}
		if !strings.Contains(err.Error(), "empty database connection string") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Invalid config connection string", func(t *testing.T) {
		_, err := database.InitDB(ctx, "invalid_conn_scheme://:")
		if err == nil {
			t.Fatal("expected error for invalid connection string, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse conn config") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Unreachable host connection string", func(t *testing.T) {
		unreachableConn := "postgres://user:pass@127.0.0.1:1/dbname?connect_timeout=1"
		_, err := database.InitDB(ctx, unreachableConn)
		if err == nil {
			t.Fatal("expected error for unreachable database, got nil")
		}
		if !strings.Contains(err.Error(), "failed to connect to database") &&
			!strings.Contains(err.Error(), "failed to ping database") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Invalid port connection string", func(t *testing.T) {
		_, err := database.InitDB(ctx, "postgres://user:pass@localhost:invalidport/dbname")
		if err == nil {
			t.Fatal("expected error for invalid port connection string, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse conn config") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestRunMigrations_OfflinePoolError(t *testing.T) {
	config, err := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:1/dbname")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err = database.RunMigrations(ctx, pool)
	if err == nil {
		t.Fatal("expected error executing migration with canceled context, got nil")
	}
	if !strings.Contains(err.Error(), "failed to execute migration") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitDB_SuccessAndMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping integration test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping initialized database: %v", err)
	}

	// Verify tables were created by migrations
	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'files');").Scan(&exists)
	if err != nil {
		t.Fatalf("failed to query table existence: %v", err)
	}
	if !exists {
		t.Error("expected 'files' table to exist after migrations")
	}

	// Verify 000002_auth_schema.sql tables and columns exist
	var userSessionsExists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'user_sessions');").Scan(&userSessionsExists)
	if err != nil || !userSessionsExists {
		t.Errorf("expected 'user_sessions' table to exist after migrations, err: %v", err)
	}

	var hasPasswordHash bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'password_hash');").Scan(&hasPasswordHash)
	if err != nil || !hasPasswordHash {
		t.Errorf("expected 'password_hash' column in 'users' table after migrations, err: %v", err)
	}

	t.Run("RunMigrations with cancelled context returns error", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := database.RunMigrations(canceledCtx, pool)
		if err == nil {
			t.Error("expected error for cancelled context in RunMigrations, got nil")
		}
	})

	t.Run("InitDB context timeout/cancellation during migration closes pool", func(t *testing.T) {
		connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
		for i := 1; i <= 20; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			go func(d time.Duration) {
				time.Sleep(d)
				cancel()
			}(time.Duration(i*200) * time.Microsecond)
			pool, err := database.InitDB(ctx, connStr)
			if err == nil {
				pool.Close()
			}
		}
	})
}

func TestBuildDSN(t *testing.T) {
	t.Run("Escapes special characters in password and sets connection params", func(t *testing.T) {
		user := "cloud_user"
		pass := "p@ss:w/ord%123"
		host := "localhost"
		port := "5432"
		dbname := "simplecloud"
		sslmode := "disable"

		dsn := database.BuildDSN(user, pass, host, port, dbname, sslmode)
		expected := "postgres://cloud_user:p%40ss%3Aw%2Ford%25123@localhost:5432/simplecloud?sslmode=disable"
		if dsn != expected {
			t.Errorf("expected DSN %q, got %q", expected, dsn)
		}
	})

	t.Run("Standard credentials without special characters", func(t *testing.T) {
		dsn := database.BuildDSN("user", "pass", "127.0.0.1", "5432", "mydb", "disable")
		expected := "postgres://user:pass@127.0.0.1:5432/mydb?sslmode=disable"
		if dsn != expected {
			t.Errorf("expected DSN %q, got %q", expected, dsn)
		}
	})
}

func TestSchemaMigrationsTable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping migration tracking integration test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	var tableExists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'schema_migrations');").Scan(&tableExists)
	if err != nil || !tableExists {
		t.Fatalf("expected 'schema_migrations' table to exist after RunMigrations, err: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations;").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows in schema_migrations: %v", err)
	}
	if count == 0 {
		t.Errorf("expected at least 1 migration recorded in schema_migrations table, got 0")
	}

	err = database.RunMigrations(ctx, pool)
	if err != nil {
		t.Fatalf("re-running RunMigrations failed: %v", err)
	}

	var countAfter int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations;").Scan(&countAfter)
	if err != nil {
		t.Fatalf("failed to count rows in schema_migrations after rerun: %v", err)
	}
	if countAfter != count {
		t.Errorf("expected count of schema_migrations to stay at %d on rerun, got %d", count, countAfter)
	}
}
