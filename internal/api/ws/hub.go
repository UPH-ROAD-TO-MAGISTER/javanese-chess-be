package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: map[string]map[*websocket.Conn]struct{}{},
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Hub) HandleWS(c *gin.Context) {
	roomCode := c.Query("roomCode")
	if roomCode == "" {
		c.String(http.StatusBadRequest, "roomCode required")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}

	h.mu.Lock()
	if _, ok := h.rooms[roomCode]; !ok {
		h.rooms[roomCode] = map[*websocket.Conn]struct{}{}
	}
	h.rooms[roomCode][conn] = struct{}{}
	h.mu.Unlock()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.rooms[roomCode], conn)
	h.mu.Unlock()
	_ = conn.Close()
}

func (h *Hub) Broadcast(roomCode string, event string, payload any) {
	msg := map[string]any{
		"type": event,
		"data": payload,
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if roomCode == "ALL" {
		for rc := range h.rooms {
			for conn := range h.rooms[rc] {
				_ = conn.WriteJSON(msg)
			}
		}
		return
	}
	for conn := range h.rooms[roomCode] {
		_ = conn.WriteJSON(msg)
	}
}
