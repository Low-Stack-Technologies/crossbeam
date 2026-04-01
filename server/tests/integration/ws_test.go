package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/low-stack-technologies/crossbeam/server/internal/ws"
	"github.com/low-stack-technologies/crossbeam/server/tests/helpers"
)

func setupWSEnv(t *testing.T) (httpSrv *httptest.Server, wsBase string, token string) {
	t.Helper()
	pool, q := helpers.NewTestSetup(t)
	helpers.Truncate(t, pool)
	redisClients := helpers.NewTestRedis(t)
	authSvc := helpers.NewTestAuthService(q)
	deviceSvc := helpers.NewTestDeviceService(q, authSvc, redisClients)
	pushSvc := helpers.NewTestPushService(q, redisClients)
	gateway := helpers.NewTestGateway(authSvc, deviceSvc, pushSvc, redisClients)
	httpSrv = helpers.NewTestServerWithGateway(t, authSvc, deviceSvc, pushSvc, gateway, q)

	resp := postJSON(t, httpSrv.URL+"/api/v1/auth/register", map[string]string{
		"email": "wstest@example.com", "password": "password123", "name": "WS Tester",
	})
	defer resp.Body.Close()
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	token = body["token"].(string)

	wsBase = "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	return
}

func dialWS(t *testing.T, wsBase, token string) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(wsBase+"/gateway?token="+token, nil)
	require.NoError(t, err)
	if resp != nil {
		resp.Body.Close()
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func nextWSEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second)) //nolint:errcheck
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	var event map[string]any
	require.NoError(t, json.Unmarshal(msg, &event))
	return event
}

func TestWS_Connect_ReceivesReady(t *testing.T) {
	_, wsBase, token := setupWSEnv(t)
	conn := dialWS(t, wsBase, token)
	event := nextWSEvent(t, conn)
	assert.Equal(t, ws.OpReady, event["op"])
}

func TestWS_InvalidToken_Rejected(t *testing.T) {
	_, wsBase, _ := setupWSEnv(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsBase+"/gateway?token=bad.token.here", nil)
	if resp != nil {
		resp.Body.Close()
	}
	assert.Error(t, err)
}

func TestWS_MissingToken_Rejected(t *testing.T) {
	_, wsBase, _ := setupWSEnv(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsBase+"/gateway", nil)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	}
	assert.Error(t, err)
}
