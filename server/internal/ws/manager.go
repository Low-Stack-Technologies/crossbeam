package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Manager tracks active WebSocket connections per user and device.
// It is safe for concurrent use.
type Manager struct {
	mu    sync.RWMutex
	conns map[uuid.UUID]map[uuid.UUID]*websocket.Conn // userID → deviceID → conn
}

func NewManager() *Manager {
	return &Manager{
		conns: make(map[uuid.UUID]map[uuid.UUID]*websocket.Conn),
	}
}

func (m *Manager) Add(userID, deviceID uuid.UUID, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conns[userID] == nil {
		m.conns[userID] = make(map[uuid.UUID]*websocket.Conn)
	}
	// close any existing connection for this device before replacing
	if old, ok := m.conns[userID][deviceID]; ok {
		old.Close() //nolint:errcheck
	}
	m.conns[userID][deviceID] = conn
}

func (m *Manager) Remove(userID, deviceID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if devices, ok := m.conns[userID]; ok {
		delete(devices, deviceID)
		if len(devices) == 0 {
			delete(m.conns, userID)
		}
	}
}

// GetByUser returns a snapshot of all connections for a user.
func (m *Manager) GetByUser(userID uuid.UUID) map[uuid.UUID]*websocket.Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	devices := m.conns[userID]
	if len(devices) == 0 {
		return nil
	}
	snapshot := make(map[uuid.UUID]*websocket.Conn, len(devices))
	for k, v := range devices {
		snapshot[k] = v
	}
	return snapshot
}

func (m *Manager) IsOnline(userID, deviceID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.conns[userID][deviceID]
	return ok
}
