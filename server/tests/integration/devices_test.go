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

func setupDeviceTest(t *testing.T) (serverURL, token string) {
	t.Helper()
	pool, q := helpers.NewTestSetup(t)
	helpers.Truncate(t, pool)
	redisClients := helpers.NewTestRedis(t)
	authSvc := helpers.NewTestAuthService(q)
	deviceSvc := helpers.NewTestDeviceService(q, authSvc, redisClients)
	pushSvc := helpers.NewTestPushService(q, redisClients)
	srv := helpers.NewTestServer(t, authSvc, deviceSvc, pushSvc, q)

	resp := postJSON(t, srv.URL+"/api/v1/auth/register", map[string]string{
		"email": "devicetest@example.com", "password": "password123", "name": "Device Tester",
	})
	defer resp.Body.Close()
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return srv.URL, body["token"].(string)
}

func postJSONAuth(t *testing.T, url string, body any, token string) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func authReq(t *testing.T, method, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestDevices_Create(t *testing.T) {
	base, token := setupDeviceTest(t)

	resp := postJSONAuth(t, base+"/api/v1/devices", map[string]string{
		"name": "My Laptop", "type": "desktop",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["token"])
	device, ok := body["device"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "My Laptop", device["name"])
}

func TestDevices_List(t *testing.T) {
	base, token := setupDeviceTest(t)

	postJSONAuth(t, base+"/api/v1/devices", map[string]string{"name": "Laptop", "type": "desktop"}, token)
	postJSONAuth(t, base+"/api/v1/devices", map[string]string{"name": "Phone", "type": "mobile"}, token)

	resp := authReq(t, http.MethodGet, base+"/api/v1/devices", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	devices, ok := body["devices"].([]any)
	require.True(t, ok)
	assert.Len(t, devices, 2)
}

func TestDevices_Delete(t *testing.T) {
	base, token := setupDeviceTest(t)

	createResp := postJSONAuth(t, base+"/api/v1/devices", map[string]string{"name": "ToDelete", "type": "desktop"}, token)
	var createBody map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&createBody))
	createResp.Body.Close()

	deviceID := createBody["device"].(map[string]any)["id"].(string)
	delResp := authReq(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/devices/%s", base, deviceID), token)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}

func TestDevices_Delete_NotOwned(t *testing.T) {
	base, token1 := setupDeviceTest(t)

	regResp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "other@example.com", "password": "password123", "name": "Other",
	})
	var regBody map[string]any
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regBody))
	regResp.Body.Close()
	token2 := regBody["token"].(string)

	createResp := postJSONAuth(t, base+"/api/v1/devices", map[string]string{"name": "Laptop", "type": "desktop"}, token1)
	var createBody map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&createBody))
	createResp.Body.Close()
	deviceID := createBody["device"].(map[string]any)["id"].(string)

	delResp := authReq(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/devices/%s", base, deviceID), token2)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, delResp.StatusCode)
}
