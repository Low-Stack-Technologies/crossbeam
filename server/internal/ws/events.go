package ws

const (
	OpReady        = "READY"
	OpPushCreate   = "PUSH_CREATE"
	OpPushDelete   = "PUSH_DELETE"
	OpDeviceUpdate = "DEVICE_UPDATE"
)

// Event is the envelope for all WebSocket messages (server → client).
type Event struct {
	Op string `json:"op"`
	D  any    `json:"d"`
}

// ReadyPayload is sent immediately after a client authenticates.
type ReadyPayload struct {
	PendingPushes any `json:"pending_pushes"`
}

// PushDeletePayload is the payload for PUSH_DELETE events.
type PushDeletePayload struct {
	ID string `json:"id"`
}

// DeviceUpdatePayload is the payload for DEVICE_UPDATE events.
type DeviceUpdatePayload struct {
	Action string `json:"action"` // "add" | "remove"
	Device any    `json:"device"`
}
