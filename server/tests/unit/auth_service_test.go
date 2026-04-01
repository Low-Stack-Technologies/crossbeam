package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/low-stack-technologies/crossbeam/server/internal/service"
	"github.com/low-stack-technologies/crossbeam/server/tests/helpers"
)

func newAuthSvc(t *testing.T) *service.AuthService {
	t.Helper()
	q := helpers.NewTestQueries(t)
	helpers.Truncate(t, helpers.NewTestPool(t))
	return helpers.NewTestAuthService(q)
}

func TestAuthService_Register(t *testing.T) {
	svc := newAuthSvc(t)
	ctx := context.Background()

	user, token, err := svc.Register(ctx, "alice@example.com", "password123", "Alice")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice", user.Name)
	assert.NotEmpty(t, user.Password) // stored as hash
	assert.NotEqual(t, "password123", user.Password)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc := newAuthSvc(t)
	ctx := context.Background()

	_, _, err := svc.Register(ctx, "dup@example.com", "password123", "First")
	require.NoError(t, err)

	_, _, err = svc.Register(ctx, "dup@example.com", "password456", "Second")
	assert.ErrorIs(t, err, service.ErrEmailTaken)
}

func TestAuthService_Login(t *testing.T) {
	svc := newAuthSvc(t)
	ctx := context.Background()

	_, _, err := svc.Register(ctx, "bob@example.com", "mypassword", "Bob")
	require.NoError(t, err)

	user, token, err := svc.Login(ctx, "bob@example.com", "mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "bob@example.com", user.Email)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc := newAuthSvc(t)
	ctx := context.Background()

	_, _, err := svc.Register(ctx, "carol@example.com", "correct", "Carol")
	require.NoError(t, err)

	_, _, err = svc.Login(ctx, "carol@example.com", "wrong")
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	svc := newAuthSvc(t)
	_, _, err := svc.Login(context.Background(), "nobody@example.com", "pass")
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
}

func TestAuthService_VerifyToken(t *testing.T) {
	svc := newAuthSvc(t)
	ctx := context.Background()

	_, token, err := svc.Register(ctx, "dave@example.com", "password123", "Dave")
	require.NoError(t, err)

	claims, err := svc.VerifyToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.Subject)
}

func TestAuthService_VerifyToken_InvalidToken(t *testing.T) {
	svc := newAuthSvc(t)
	_, err := svc.VerifyToken("invalid.token.here")
	assert.Error(t, err)
}
