package http

import (
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"

	"github.com/gin-gonic/gin"
)

func NewRouter(rm *room.Manager, cfg config.Config) *gin.Engine {
	r := gin.Default()
	hub := ws.NewHub()

	// WebSocket for FE live updates
	r.GET("/ws", hub.HandleWS)

	// --- ROOM ENDPOINTS ---
	r.POST("/create-room", CreateRoomHandler(rm))
	r.POST("/play", PlayHandler(rm, hub))

	// --- GAME ENDPOINTS ---
	r.GET("/possible-moves", PossibleMovesHandler(rm))
	r.POST("/set-hands", SetHandsHandler(rm, hub))
	r.POST("/move", MoveHandler(rm, hub))
	r.POST("/move-bot", MoveBotHandler(rm, hub))

	// --- CONFIG ENDPOINTS ---
	r.GET("/config/weights", GetConfigHandler(cfg))
	r.POST("/config/weights", UpdateConfigHandler(cfg, hub))

	return r
}
