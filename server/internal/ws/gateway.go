package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	appredis "github.com/low-stack-technologies/crossbeam/server/internal/redis"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Gateway handles WebSocket connections and Redis pub/sub fan-out.
type Gateway struct {
	manager   *Manager
	authSvc   *service.AuthService
	deviceSvc *service.DeviceService
	pushSvc   *service.PushService
	redis     *appredis.Clients
}

func NewGateway(
	manager *Manager,
	authSvc *service.AuthService,
	deviceSvc *service.DeviceService,
	pushSvc *service.PushService,
	redis *appredis.Clients,
) *Gateway {
	return &Gateway{
		manager:   manager,
		authSvc:   authSvc,
		deviceSvc: deviceSvc,
		pushSvc:   pushSvc,
		redis:     redis,
	}
}

// Start launches the Redis subscriber goroutine. Call once at startup.
func (g *Gateway) Start(ctx context.Context) {
	go g.subscribeRedis(ctx)
}

// ServeHTTP upgrades the HTTP connection and handles the WebSocket lifecycle.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := g.authSvc.VerifyToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		http.Error(w, "invalid token subject", http.StatusUnauthorized)
		return
	}

	var deviceID uuid.UUID
	if claims.DeviceID != nil {
		deviceID = *claims.DeviceID
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "error", err)
		return
	}

	g.manager.Add(userID, deviceID, conn)
	defer func() {
		g.manager.Remove(userID, deviceID)
		conn.Close() //nolint:errcheck
		if deviceID != (uuid.UUID{}) {
			g.deviceSvc.UpdateLastSeen(context.Background(), deviceID) //nolint:errcheck
		}
	}()

	if err := g.sendReady(conn, userID, deviceID); err != nil {
		slog.Warn("ws send ready failed", "error", err)
		return
	}

	// Read loop: keeps conn alive and detects client disconnect.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (g *Gateway) sendReady(conn *websocket.Conn, userID, deviceID uuid.UUID) error {
	ctx := context.Background()

	pending := []any{}
	if deviceID != (uuid.UUID{}) {
		pushes, err := g.pushSvc.GetPendingPushes(ctx, deviceID, userID)
		if err == nil {
			for i := range pushes {
				pending = append(pending, pushes[i])
			}
		}
	}

	return conn.WriteJSON(Event{
		Op: OpReady,
		D:  ReadyPayload{PendingPushes: pending},
	})
}

func (g *Gateway) subscribeRedis(ctx context.Context) {
	sub := g.redis.Sub.PSubscribe(ctx, "user:*")
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.Channel():
			if !ok {
				return
			}
			// Channel is "user:{uuid}" — extract the userID.
			parts := strings.SplitN(msg.Channel, ":", 2)
			if len(parts) != 2 {
				continue
			}
			userID, err := uuid.Parse(parts[1])
			if err != nil {
				continue
			}
			g.SendToUser(userID, json.RawMessage(msg.Payload))
		}
	}
}

// SendToUser delivers a raw JSON event to all connections for a specific user.
func (g *Gateway) SendToUser(userID uuid.UUID, raw json.RawMessage) {
	conns := g.manager.GetByUser(userID)
	for _, conn := range conns {
		conn.WriteMessage(websocket.TextMessage, raw) //nolint:errcheck
	}
}
