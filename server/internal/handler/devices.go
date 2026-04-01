package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/middleware"
	"github.com/low-stack-technologies/crossbeam/server/internal/service"
)

type DeviceHandler struct {
	deviceSvc *service.DeviceService
}

func NewDeviceHandler(deviceSvc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceSvc: deviceSvc}
}

type deviceResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	LastSeen  *time.Time `json:"last_seen"`
	CreatedAt time.Time  `json:"created_at"`
}

func toDeviceResponse(d *generated.Device) deviceResponse {
	var lastSeen *time.Time
	if d.LastSeen.Valid {
		t := d.LastSeen.Time
		lastSeen = &t
	}
	return deviceResponse{
		ID:        db.PgToUUID(d.ID),
		UserID:    db.PgToUUID(d.UserID),
		Name:      d.Name,
		Type:      d.Type,
		LastSeen:  lastSeen,
		CreatedAt: d.CreatedAt.Time,
	}
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	devices, err := h.deviceSvc.ListDevices(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	resp := make([]deviceResponse, len(devices))
	for i := range devices {
		resp[i] = toDeviceResponse(&devices[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": resp})
}

type registerDeviceRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" {
		writeError(w, http.StatusUnprocessableEntity, "name and type are required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	device, token, err := h.deviceSvc.RegisterDevice(r.Context(), userID, req.Name, req.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register device")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"device": toDeviceResponse(device),
		"token":  token,
	})
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	deviceIDStr := chi.URLParam(r, "id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id")
		return
	}

	if err := h.deviceSvc.DeleteDevice(r.Context(), userID, deviceID); err != nil {
		if errors.Is(err, service.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

