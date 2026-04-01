package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/low-stack-technologies/crossbeam/server/tests/helpers"
)

func setupAuthTest(t *testing.T) (serverURL string) {
	t.Helper()
	pool, q := helpers.NewTestSetup(t)
	helpers.Truncate(t, pool)
	redisClients := helpers.NewTestRedis(t)
	authSvc := helpers.NewTestAuthService(q)
	deviceSvc := helpers.NewTestDeviceService(q, authSvc, redisClients)
	pushSvc := helpers.NewTestPushService(q, redisClients)
	srv := helpers.NewTestServer(t, authSvc, deviceSvc, pushSvc, q)
	return srv.URL
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b)) //nolint:noctx
	require.NoError(t, err)
	return resp
}

func TestRegister_Success(t *testing.T) {
	base := setupAuthTest(t)

	resp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
		"name":     "Alice",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["token"])
	user, ok := body["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", user["email"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	base := setupAuthTest(t)

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "dup@example.com", "password": "password123", "name": "First",
	})

	resp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "dup@example.com", "password": "password456", "name": "Second",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRegister_MissingFields(t *testing.T) {
	base := setupAuthTest(t)

	resp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "missing@example.com",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestLogin_Success(t *testing.T) {
	base := setupAuthTest(t)

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "bob@example.com", "password": "mypassword", "name": "Bob",
	})

	resp := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email": "bob@example.com", "password": "mypassword",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	base := setupAuthTest(t)

	postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "carol@example.com", "password": "correct", "name": "Carol",
	})

	resp := postJSON(t, base+"/api/v1/auth/login", map[string]string{
		"email": "carol@example.com", "password": "wrong",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetMe_Success(t *testing.T) {
	base := setupAuthTest(t)

	regResp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "dave@example.com", "password": "password123", "name": "Dave",
	})
	defer regResp.Body.Close()

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regBody))
	token := regBody["token"].(string)

	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/users/@me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "dave@example.com", body["email"])
}

func TestGetMe_Unauthorized(t *testing.T) {
	base := setupAuthTest(t)

	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/users/@me", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
