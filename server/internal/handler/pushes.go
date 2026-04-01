package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/middleware"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
	"github.com/low-stack-technologies/crossbeam/server/internal/storage"
)

type PushHandler struct {
	pushSvc    *service.PushService
	storageSvc *storage.Client // optional; nil disables file uploads
}

func NewPushHandler(pushSvc *service.PushService, storageSvc *storage.Client) *PushHandler {
	return &PushHandler{pushSvc: pushSvc, storageSvc: storageSvc}
}

type pushResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	SourceDeviceID *uuid.UUID `json:"source_device_id,omitempty"`
	TargetDeviceID *uuid.UUID `json:"target_device_id,omitempty"`
	Type           string     `json:"type"`
	Title          *string    `json:"title,omitempty"`
	Body           *string    `json:"body,omitempty"`
	URL            *string    `json:"url,omitempty"`
	FileName       *string    `json:"file_name,omitempty"`
	FileType       *string    `json:"file_type,omitempty"`
	FileSize       *int64     `json:"file_size,omitempty"`
	FileURL        *string    `json:"file_url,omitempty"`
	Delivered      bool       `json:"delivered"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toPushResponse(p *generated.Push) pushResponse {
	return pushResponse{
		ID:             db.PgToUUID(p.ID),
		UserID:         db.PgToUUID(p.UserID),
		SourceDeviceID: db.PgToUUIDPtr(p.SourceDeviceID),
		TargetDeviceID: db.PgToUUIDPtr(p.TargetDeviceID),
		Type:           p.Type,
		Title:          p.Title,
		Body:           p.Body,
		URL:            p.Url,
		FileName:       p.FileName,
		FileType:       p.FileType,
		FileSize:       p.FileSize,
		Delivered:      p.Delivered,
		CreatedAt:      p.CreatedAt.Time,
	}
}

func (h *PushHandler) enrichWithFileURL(ctx context.Context, resp *pushResponse, p *generated.Push) {
	if h.storageSvc == nil || p.FileS3Key == nil {
		return
	}
	url, err := h.storageSvc.GetPresignedURL(ctx, *p.FileS3Key, 15*time.Minute)
	if err == nil {
		resp.FileURL = &url
	}
}

const defaultPushLimit = 50
const maxPushLimit = 100

func (h *PushHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	cursor := time.Now()
	if c := r.URL.Query().Get("cursor"); c != "" {
		if t, err := time.Parse(time.RFC3339Nano, c); err == nil {
			cursor = t
		}
	}

	limit := defaultPushLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= maxPushLimit {
			limit = n
		}
	}

	pushes, err := h.pushSvc.ListPushes(r.Context(), userID, cursor, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pushes")
		return
	}

	resp := make([]pushResponse, len(pushes))
	for i := range pushes {
		pr := toPushResponse(&pushes[i])
		h.enrichWithFileURL(r.Context(), &pr, &pushes[i])
		resp[i] = pr
	}

	var nextCursor *string
	if len(pushes) == limit {
		t := resp[len(resp)-1].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &t
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pushes":      resp,
		"next_cursor": nextCursor,
	})
}

type createPushRequest struct {
	TargetDeviceID *string `json:"target_device_id"`
	Type           string  `json:"type"`
	Title          *string `json:"title"`
	Body           *string `json:"body"`
	URL            *string `json:"url"`
}

const maxUploadSize = 25 << 20 // 25 MB

func (h *PushHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deviceID := middleware.GetDeviceID(r.Context())

	var params service.CreatePushParams

	contentType := r.Header.Get("Content-Type")
	if len(contentType) >= 9 && contentType[:9] == "multipart" {
		if h.storageSvc == nil {
			writeError(w, http.StatusServiceUnavailable, "file uploads not configured")
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		params.Type = r.FormValue("type")
		if title := r.FormValue("title"); title != "" {
			params.Title = &title
		}
		if body := r.FormValue("body"); body != "" {
			params.Body = &body
		}
		if targetIDStr := r.FormValue("target_device_id"); targetIDStr != "" {
			id, err := uuid.Parse(targetIDStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid target_device_id")
				return
			}
			params.TargetDeviceID = &id
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()

		fileName := header.Filename
		fileType := header.Header.Get("Content-Type")
		fileSize := header.Size
		key := h.storageSvc.GenerateKey(userID.String(), fileName)

		if err := h.storageSvc.UploadFile(r.Context(), key, file, fileType, fileSize); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to upload file")
			return
		}

		params.FileName = &fileName
		params.FileType = &fileType
		params.FileS3Key = &key
		params.FileSize = &fileSize
	} else {
		var req createPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		params.Type = req.Type
		params.Title = req.Title
		params.Body = req.Body
		params.URL = req.URL
		if req.TargetDeviceID != nil {
			id, err := uuid.Parse(*req.TargetDeviceID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid target_device_id")
				return
			}
			params.TargetDeviceID = &id
		}
	}

	if params.Type != "note" && params.Type != "link" && params.Type != "file" {
		writeError(w, http.StatusUnprocessableEntity, "type must be note, link, or file")
		return
	}

	if deviceID != (uuid.UUID{}) {
		params.SourceDeviceID = &deviceID
	}

	push, err := h.pushSvc.CreatePush(r.Context(), userID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create push")
		return
	}

	pr := toPushResponse(push)
	h.enrichWithFileURL(r.Context(), &pr, push)
	writeJSON(w, http.StatusCreated, map[string]any{"push": pr})
}

func (h *PushHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	pushIDStr := chi.URLParam(r, "id")
	pushID, err := uuid.Parse(pushIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid push id")
		return
	}

	if err := h.pushSvc.DeletePush(r.Context(), userID, pushID); err != nil {
		if errors.Is(err, service.ErrPushNotFound) {
			writeError(w, http.StatusNotFound, "push not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete push")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
