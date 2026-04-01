package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	appredis "github.com/low-stack-technologies/crossbeam/server/internal/redis"
	"github.com/low-stack-technologies/crossbeam/server/internal/storage"
)

var ErrPushNotFound = fmt.Errorf("push not found")

type CreatePushParams struct {
	SourceDeviceID *uuid.UUID
	TargetDeviceID *uuid.UUID // nil = broadcast
	Type           string
	Title          *string
	Body           *string
	URL            *string
	FileName       *string
	FileType       *string
	FileS3Key      *string
	FileSize       *int64
}

type PushService struct {
	q       *generated.Queries
	redis   *appredis.Clients
	storage *storage.Client // optional; nil skips file deletion
}

func NewPushService(q *generated.Queries, redis *appredis.Clients, storageSvc *storage.Client) *PushService {
	return &PushService{q: q, redis: redis, storage: storageSvc}
}

func (s *PushService) ListPushes(ctx context.Context, userID uuid.UUID, cursor time.Time, limit int) ([]generated.Push, error) {
	pushes, err := s.q.ListPushes(ctx, generated.ListPushesParams{
		UserID:    db.UUIDToPg(userID),
		CreatedAt: db.TimeToPg(cursor),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list pushes: %w", err)
	}
	return pushes, nil
}

func (s *PushService) CreatePush(ctx context.Context, userID uuid.UUID, params CreatePushParams) (*generated.Push, error) {
	push, err := s.q.CreatePush(ctx, generated.CreatePushParams{
		UserID:         db.UUIDToPg(userID),
		SourceDeviceID: db.UUIDPtrToPg(params.SourceDeviceID),
		TargetDeviceID: db.UUIDPtrToPg(params.TargetDeviceID),
		Type:           params.Type,
		Title:          params.Title,
		Body:           params.Body,
		Url:            params.URL,
		FileName:       params.FileName,
		FileType:       params.FileType,
		FileS3Key:      params.FileS3Key,
		FileSize:       params.FileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("create push: %w", err)
	}

	s.fanOut(ctx, userID, &push)
	return &push, nil
}

func (s *PushService) DeletePush(ctx context.Context, userID, pushID uuid.UUID) error {
	push, err := s.q.GetPushByID(ctx, generated.GetPushByIDParams{
		ID:     db.UUIDToPg(pushID),
		UserID: db.UUIDToPg(userID),
	})
	if err != nil {
		return ErrPushNotFound
	}

	if err := s.q.DeletePush(ctx, generated.DeletePushParams{
		ID:     db.UUIDToPg(pushID),
		UserID: db.UUIDToPg(userID),
	}); err != nil {
		return fmt.Errorf("delete push: %w", err)
	}

	if s.storage != nil && push.FileS3Key != nil {
		s.storage.DeleteFile(ctx, *push.FileS3Key) //nolint:errcheck
	}

	s.publishPushDelete(ctx, userID, db.PgToUUID(push.ID))
	return nil
}

func (s *PushService) GetPendingPushes(ctx context.Context, deviceID, userID uuid.UUID) ([]generated.Push, error) {
	pushes, err := s.q.GetPendingPushes(ctx, generated.GetPendingPushesParams{
		TargetDeviceID: db.UUIDToPg(deviceID),
		UserID:         db.UUIDToPg(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("get pending pushes: %w", err)
	}
	return pushes, nil
}

func (s *PushService) fanOut(ctx context.Context, userID uuid.UUID, push *generated.Push) {
	if err := s.q.MarkPushDelivered(ctx, push.ID); err != nil {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"op": "PUSH_CREATE",
		"d":  push,
	})
	if err != nil {
		return
	}
	s.redis.Pub.Publish(ctx, userChannel(userID), payload) //nolint:errcheck
}

func (s *PushService) publishPushDelete(ctx context.Context, userID, pushID uuid.UUID) {
	payload, err := json.Marshal(map[string]any{
		"op": "PUSH_DELETE",
		"d":  map[string]string{"id": pushID.String()},
	})
	if err != nil {
		return
	}
	s.redis.Pub.Publish(ctx, userChannel(userID), payload) //nolint:errcheck
}
