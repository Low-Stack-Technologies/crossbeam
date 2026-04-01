package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/service"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "userID"
	ContextKeyDeviceID contextKey = "deviceID"
)

func Auth(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			claims, err := authSvc.VerifyToken(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token subject")
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			if claims.DeviceID != nil {
				ctx = context.WithValue(ctx, ContextKeyDeviceID, *claims.DeviceID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return v
}

func GetDeviceID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyDeviceID).(uuid.UUID)
	return v
}
