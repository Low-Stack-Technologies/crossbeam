package helpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/low-stack-technologies/crossbeam/server/internal/config"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/handler"
	appmiddleware "github.com/low-stack-technologies/crossbeam/server/internal/middleware"
	appredis "github.com/low-stack-technologies/crossbeam/server/internal/redis"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
	"github.com/low-stack-technologies/crossbeam/server/internal/storage"
	"github.com/low-stack-technologies/crossbeam/server/internal/ws"
)

const TestJWTSecret = "test-secret-key-for-testing-only"

// NewTestAuthService creates an AuthService with a fixed test secret.
func NewTestAuthService(q *generated.Queries) *service.AuthService {
	return service.NewAuthService(q, TestJWTSecret, 24*time.Hour)
}

// NewTestDeviceService creates a DeviceService wired with the given clients.
func NewTestDeviceService(q *generated.Queries, authSvc *service.AuthService, redis *appredis.Clients) *service.DeviceService {
	return service.NewDeviceService(q, authSvc, redis)
}

// NewTestServer builds a fully-wired httptest.Server for integration tests.
func NewTestServer(t *testing.T, authSvc *service.AuthService, deviceSvc *service.DeviceService, pushSvc *service.PushService, q *generated.Queries) *httptest.Server {
	t.Helper()

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(q)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)
	pushHandler := handler.NewPushHandler(pushSvc, nil)
	authMiddleware := appmiddleware.Auth(authSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// NewTestPushService creates a PushService for tests (no file storage).
func NewTestPushService(q *generated.Queries, redis *appredis.Clients) *service.PushService {
	return service.NewPushService(q, redis, nil)
}

// NewTestGateway creates a WebSocket gateway for tests.
func NewTestGateway(authSvc *service.AuthService, deviceSvc *service.DeviceService, pushSvc *service.PushService, redis *appredis.Clients) *ws.Gateway {
	manager := ws.NewManager()
	gateway := ws.NewGateway(manager, authSvc, deviceSvc, pushSvc, redis)
	gateway.Start(context.Background())
	return gateway
}

// NewTestServerWithGateway builds a test server that includes the WS gateway.
func NewTestServerWithGateway(t *testing.T, authSvc *service.AuthService, deviceSvc *service.DeviceService, pushSvc *service.PushService, gateway *ws.Gateway, q *generated.Queries) *httptest.Server {
	t.Helper()

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(q)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)
	pushHandler := handler.NewPushHandler(pushSvc, nil)
	authMiddleware := appmiddleware.Auth(authSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)

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

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// NewTestStorage creates a storage client pointing at local MinIO; skips if unavailable.
func NewTestStorage(t *testing.T) *storage.Client {
	t.Helper()
	cfg := &config.Config{
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		S3AccessKey: "minioadmin",
		S3SecretKey: "minioadmin",
		S3Bucket:    "crossbeam-test",
	}
	client := storage.New(cfg)
	if err := client.EnsureBucket(context.Background()); err != nil {
		t.Skipf("minio unavailable (%v) — skipping test", err)
	}
	return client
}

// NewTestPushServiceWithStorage creates a PushService with a real storage client.
func NewTestPushServiceWithStorage(q *generated.Queries, redis *appredis.Clients, storageSvc *storage.Client) *service.PushService {
	return service.NewPushService(q, redis, storageSvc)
}

// NewTestServerFull builds a test server wired with a real storage client.
func NewTestServerFull(t *testing.T, authSvc *service.AuthService, deviceSvc *service.DeviceService, pushSvc *service.PushService, storageSvc *storage.Client, q *generated.Queries) *httptest.Server {
	t.Helper()

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(q)
	deviceHandler := handler.NewDeviceHandler(deviceSvc)
	pushHandler := handler.NewPushHandler(pushSvc, storageSvc)
	authMiddleware := appmiddleware.Auth(authSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)

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

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// NewTestRedis creates a no-op Redis clients stub for tests that don't need real Redis.
func NewTestRedis(t *testing.T) *appredis.Clients {
	t.Helper()
	redisURL := "redis://localhost:6379"
	clients, err := appredis.New(redisURL)
	if err != nil {
		t.Skipf("redis unavailable (%v) — skipping test", err)
	}
	t.Cleanup(func() {
		clients.Pub.Close()
		clients.Sub.Close()
	})
	return clients
}
