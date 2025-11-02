package store

import (
	"javanese-chess/internal/shared"
	"sync"
)

type MemoryStore struct {
	mu    sync.RWMutex
	rooms map[string]*shared.Room
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms: map[string]*shared.Room{},
	}
}

func (m *MemoryStore) GetRoom(code string) (*shared.Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	return r, ok
}

func (m *MemoryStore) SaveRoom(r *shared.Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[r.Code] = r
}
