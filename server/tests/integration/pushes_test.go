package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/low-stack-technologies/crossbeam/server/tests/helpers"
)

func setupPushTest(t *testing.T) (serverURL, token string) {
	t.Helper()
	pool, q := helpers.NewTestSetup(t)
	helpers.Truncate(t, pool)
	redisClients := helpers.NewTestRedis(t)
	authSvc := helpers.NewTestAuthService(q)
	deviceSvc := helpers.NewTestDeviceService(q, authSvc, redisClients)
	pushSvc := helpers.NewTestPushService(q, redisClients)
	srv := helpers.NewTestServer(t, authSvc, deviceSvc, pushSvc, q)

	resp := postJSON(t, srv.URL+"/api/v1/auth/register", map[string]string{
		"email": "pushtest@example.com", "password": "password123", "name": "Push Tester",
	})
	defer resp.Body.Close()
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return srv.URL, body["token"].(string)
}

func TestPushes_CreateNote(t *testing.T) {
	base, token := setupPushTest(t)

	body := "Hello from the other side"
	resp := postJSONAuth(t, base+"/api/v1/pushes", map[string]any{
		"type": "note",
		"body": body,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	push, ok := respBody["push"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "note", push["type"])
	assert.Equal(t, body, push["body"])
}

func TestPushes_CreateLink(t *testing.T) {
	base, token := setupPushTest(t)

	resp := postJSONAuth(t, base+"/api/v1/pushes", map[string]any{
		"type":  "link",
		"title": "Golang website",
		"url":   "https://go.dev",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	push := respBody["push"].(map[string]any)
	assert.Equal(t, "link", push["type"])
	assert.Equal(t, "https://go.dev", push["url"])
}

func TestPushes_List(t *testing.T) {
	base, token := setupPushTest(t)

	postJSONAuth(t, base+"/api/v1/pushes", map[string]any{"type": "note", "body": "first"}, token)
	postJSONAuth(t, base+"/api/v1/pushes", map[string]any{"type": "note", "body": "second"}, token)

	resp := authReq(t, http.MethodGet, base+"/api/v1/pushes", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	pushes, ok := respBody["pushes"].([]any)
	require.True(t, ok)
	assert.Len(t, pushes, 2)
}

func TestPushes_Delete(t *testing.T) {
	base, token := setupPushTest(t)

	createResp := postJSONAuth(t, base+"/api/v1/pushes", map[string]any{"type": "note", "body": "to delete"}, token)
	var createBody map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&createBody))
	createResp.Body.Close()
	pushID := createBody["push"].(map[string]any)["id"].(string)

	delResp := authReq(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/pushes/%s", base, pushID), token)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}

func TestPushes_CreateFile(t *testing.T) {
	pool, q := helpers.NewTestSetup(t)
	helpers.Truncate(t, pool)
	redisClients := helpers.NewTestRedis(t)
	storageSvc := helpers.NewTestStorage(t)
	authSvc := helpers.NewTestAuthService(q)
	deviceSvc := helpers.NewTestDeviceService(q, authSvc, redisClients)
	pushSvc := helpers.NewTestPushServiceWithStorage(q, redisClients, storageSvc)
	srv := helpers.NewTestServerFull(t, authSvc, deviceSvc, pushSvc, storageSvc, q)

	regResp := postJSON(t, srv.URL+"/api/v1/auth/register", map[string]string{
		"email": "filetest@example.com", "password": "password123", "name": "File Tester",
	})
	var regBody map[string]any
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regBody))
	regResp.Body.Close()
	token := regBody["token"].(string)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("type", "file")          //nolint:errcheck
	mw.WriteField("title", "test file")    //nolint:errcheck
	fw, err := mw.CreateFormFile("file", "hello.txt")
	require.NoError(t, err)
	io.WriteString(fw, "hello world")      //nolint:errcheck
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/pushes", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	push := respBody["push"].(map[string]any)
	assert.Equal(t, "file", push["type"])
	assert.Equal(t, "hello.txt", push["file_name"])
	assert.NotEmpty(t, push["file_url"])
}

func TestPushes_Delete_NotOwned(t *testing.T) {
	base, token1 := setupPushTest(t)

	regResp := postJSON(t, base+"/api/v1/auth/register", map[string]string{
		"email": "other2@example.com", "password": "password123", "name": "Other2",
	})
	var regBody map[string]any
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regBody))
	regResp.Body.Close()
	token2 := regBody["token"].(string)

	createResp := postJSONAuth(t, base+"/api/v1/pushes", map[string]any{"type": "note", "body": "owned"}, token1)
	var createBody map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&createBody))
	createResp.Body.Close()
	pushID := createBody["push"].(map[string]any)["id"].(string)

	delResp := authReq(t, http.MethodDelete, fmt.Sprintf("%s/api/v1/pushes/%s", base, pushID), token2)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, delResp.StatusCode)
}
