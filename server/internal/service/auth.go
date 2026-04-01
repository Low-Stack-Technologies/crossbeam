package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrInvalidCreds = errors.New("invalid email or password")
)

// Argon2id parameters (OWASP recommended minimums)
const (
	argon2Memory      = 64 * 1024 // 64 MB
	argon2Iterations  = 1
	argon2Parallelism = 4
	argon2SaltLen     = 16
	argon2KeyLen      = 32
)

type Claims struct {
	jwt.RegisteredClaims
	DeviceID *uuid.UUID `json:"device_id,omitempty"`
}

type AuthService struct {
	q         *generated.Queries
	jwtSecret []byte
	jwtExpiry time.Duration
}

func NewAuthService(q *generated.Queries, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		q:         q,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (*generated.User, string, error) {
	if _, err := s.q.GetUserByEmail(ctx, email); err == nil {
		return nil, "", ErrEmailTaken
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.q.CreateUser(ctx, generated.CreateUserParams{
		Email:    email,
		Password: hash,
		Name:     name,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.signToken(db.PgToUUID(user.ID), nil)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*generated.User, string, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", ErrInvalidCreds
	}

	if err := verifyPassword(password, user.Password); err != nil {
		return nil, "", ErrInvalidCreds
	}

	token, err := s.signToken(db.PgToUUID(user.ID), nil)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *AuthService) VerifyToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *AuthService) SignDeviceToken(userID, deviceID uuid.UUID) (string, error) {
	return s.signToken(userID, &deviceID)
}

func (s *AuthService) signToken(userID uuid.UUID, deviceID *uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		DeviceID: deviceID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)
	return base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}

func verifyPassword(password, encoded string) error {
	parts := strings.SplitN(encoded, "$", 2)
	if len(parts) != 2 {
		return errors.New("invalid hash format")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLen)
	if !bytes.Equal(hash, expected) {
		return errors.New("password mismatch")
	}
	return nil
}
