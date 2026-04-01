package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	appconfig "github.com/low-stack-technologies/crossbeam/server/internal/config"
	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/handler"
	appmiddleware "github.com/low-stack-technologies/crossbeam/server/internal/middleware"
	appredis "github.com/low-stack-technologies/crossbeam/server/internal/redis"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
	"github.com/low-stack-technologies/crossbeam/server/internal/storage"
	"github.com/low-stack-technologies/crossbeam/server/internal/ws"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	jwtExpiry, err := appconfig.ParseExpiry(cfg.JWTExpiresIn)
	if err != nil {
		slog.Error("invalid JWT_EXPIRES_IN", "error", err)
		os.Exit(1)
	}

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClients, err := appredis.New(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClients.Pub.Close()
	defer redisClients.Sub.Close()

	s3Client := storage.New(cfg)
	if err := s3Client.EnsureBucket(ctx); err != nil {
		slog.Error("failed to ensure S3 bucket", "error", err)
		os.Exit(1)
	}

	q := generated.New(pool)

	authSvc := service.NewAuthService(q, cfg.JWTSecret, jwtExpiry)
	deviceSvc := service.NewDeviceService(q, authSvc, redisClients)
	pushSvc := service.NewPushService(q, redisClients, s3Client)

	wsManager := ws.NewManager()
	gateway := ws.NewGateway(wsManager, authSvc, deviceSvc, pushSvc, redisClients)
	gateway.Start(ctx)

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(q)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)
	pushHandler := handler.NewPushHandler(pushSvc, s3Client)

	authMiddleware := appmiddleware.Auth(authSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	r.Get("/gateway", gateway.ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/users/@me", userHandler.Me)

			r.Get("/devices", deviceHandler.List)
			r.Post("/devices", deviceHandler.Create)
			r.Delete("/devices/{id}", deviceHandler.Delete)

			r.Get("/pushes", pushHandler.List)
			r.Post("/pushes", pushHandler.Create)
			r.Delete("/pushes/{id}", pushHandler.Delete)
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
