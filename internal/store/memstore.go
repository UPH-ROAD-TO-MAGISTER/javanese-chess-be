package store

import (
	"javanese-chess/internal/room"
	"sync"
)

type MemoryStore struct {
	mu    sync.RWMutex
	rooms map[string]*room.Room
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms: map[string]*room.Room{},
	}
}

func (m *MemoryStore) GetRoom(code string) (*room.Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	return r, ok
}

func (m *MemoryStore) SaveRoom(r *room.Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[r.Code] = r
}
