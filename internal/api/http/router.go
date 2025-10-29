package http

import (
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/room"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(mgr *room.Manager, s room.Store, hub *ws.Hub) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://98.70.41.170:5173", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// Existing handlers (not using store directly)
	// Room management
	r.POST("/create-room", CreateRoomHandler(mgr))
	r.POST("/play", PlayHandler(mgr, hub))

	// Game actions
	r.GET("/possible-moves", PossibleMovesHandler(mgr))
	r.POST("/set-hands", SetHandsHandler(mgr, hub))
	r.POST("/move", MoveHandler(mgr, hub))
	r.POST("/move-bot", MoveBotHandler(mgr, hub))

	// Config routes (room-based)
	configHandler := NewConfigHandler(s, hub)
	configGroup := r.Group("/config")
	{
		configGroup.GET("/weights/default", configHandler.GetDefaultWeightsHandler)
		configGroup.GET("/weights/room", configHandler.GetRoomWeightsHandler)
		configGroup.POST("/weights/room", configHandler.UpdateRoomWeightsHandler)
		configGroup.POST("/weights/room/reset", configHandler.ResetRoomWeightsHandler)
	}

	// WebSocket
	r.GET("/ws", hub.HandleWS)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
