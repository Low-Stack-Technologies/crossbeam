package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
)

const TestPassword = "testpassword123"

// SeedUser creates a user in the DB and returns the user + auth service (for token signing).
func SeedUser(t *testing.T, pool *pgxpool.Pool, authSvc *service.AuthService, email string) (*generated.User, string) {
	t.Helper()
	user, token, err := authSvc.Register(context.Background(), email, TestPassword, "Test User")
	if err != nil {
		t.Fatalf("seed user %q: %v", email, err)
	}
	return user, token
}

// SeedUserN seeds N users with auto-generated emails.
func SeedUserN(t *testing.T, pool *pgxpool.Pool, authSvc *service.AuthService, n int) ([]*generated.User, []string) {
	t.Helper()
	users := make([]*generated.User, n)
	tokens := make([]string, n)
	for i := range n {
		users[i], tokens[i] = SeedUser(t, pool, authSvc, fmt.Sprintf("user%d@example.com", i))
	}
	return users, tokens
}
