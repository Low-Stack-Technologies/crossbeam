package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	appredis "github.com/low-stack-technologies/crossbeam/server/internal/redis"
)

var ErrDeviceNotFound = fmt.Errorf("device not found")

type DeviceService struct {
	q       *generated.Queries
	authSvc *AuthService
	redis   *appredis.Clients
}

func NewDeviceService(q *generated.Queries, authSvc *AuthService, redis *appredis.Clients) *DeviceService {
	return &DeviceService{q: q, authSvc: authSvc, redis: redis}
}

func (s *DeviceService) ListDevices(ctx context.Context, userID uuid.UUID) ([]generated.Device, error) {
	devices, err := s.q.ListDevicesByUser(ctx, db.UUIDToPg(userID))
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

func (s *DeviceService) RegisterDevice(ctx context.Context, userID uuid.UUID, name, deviceType string) (*generated.Device, string, error) {
	device, err := s.q.CreateDevice(ctx, generated.CreateDeviceParams{
		UserID: db.UUIDToPg(userID),
		Name:   name,
		Type:   deviceType,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create device: %w", err)
	}

	token, err := s.authSvc.SignDeviceToken(userID, db.PgToUUID(device.ID))
	if err != nil {
		return nil, "", fmt.Errorf("sign device token: %w", err)
	}

	s.publishDeviceUpdate(ctx, userID, "add", &device)
	return &device, token, nil
}

func (s *DeviceService) DeleteDevice(ctx context.Context, userID, deviceID uuid.UUID) error {
	device, err := s.q.GetDeviceByID(ctx, db.UUIDToPg(deviceID))
	if err != nil || db.PgToUUID(device.UserID) != userID {
		return ErrDeviceNotFound
	}

	if err := s.q.DeleteDevice(ctx, generated.DeleteDeviceParams{
		ID:     db.UUIDToPg(deviceID),
		UserID: db.UUIDToPg(userID),
	}); err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	s.publishDeviceUpdate(ctx, userID, "remove", &device)
	return nil
}

func (s *DeviceService) UpdateLastSeen(ctx context.Context, deviceID uuid.UUID) error {
	return s.q.UpdateDeviceLastSeen(ctx, db.UUIDToPg(deviceID))
}

func (s *DeviceService) publishDeviceUpdate(ctx context.Context, userID uuid.UUID, action string, device *generated.Device) {
	payload, err := json.Marshal(map[string]any{
		"op": "DEVICE_UPDATE",
		"d":  map[string]any{"action": action, "device": device},
	})
	if err != nil {
		return
	}
	s.redis.Pub.Publish(ctx, userChannel(userID), payload) //nolint:errcheck
}

func userChannel(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s", userID)
}
