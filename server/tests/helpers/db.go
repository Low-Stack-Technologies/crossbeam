package helpers

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
)

func DatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return "postgresql://crossbeam:password@localhost:5432/crossbeam"
}

// NewTestPool creates a pgxpool connected to the test database and runs migrations.
func NewTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := DatabaseURL()

	if err := db.Migrate(databaseURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// NewTestSetup returns a pool and queries backed by the test database.
func NewTestSetup(t *testing.T) (*pgxpool.Pool, *generated.Queries) {
	t.Helper()
	pool := NewTestPool(t)
	return pool, generated.New(pool)
}

// NewTestQueries returns a *generated.Queries backed by a test pool.
func NewTestQueries(t *testing.T) *generated.Queries {
	t.Helper()
	pool := NewTestPool(t)
	return generated.New(pool)
}

// Truncate empties all application tables between tests.
func Truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"TRUNCATE pushes, devices, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
