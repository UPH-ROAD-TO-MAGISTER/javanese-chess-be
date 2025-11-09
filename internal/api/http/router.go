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
		AllowOrigins:     []string{"http://98.70.41.170:5000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// Existing handlers (not using store directly)
	r.POST("/api/play", PlayHandler(mgr, hub))

	// Config routes (room-based)
	configHandler := NewConfigHandler(s, hub)
	configGroup := r.Group("/api/config")
	{
		configGroup.GET("/weights/default", configHandler.GetDefaultWeightsHandler)
		configGroup.GET("/weights/room", configHandler.GetRoomWeightsHandler)
	}

	// Debug route to view logs
	r.GET("/api/debug/logs", func(c *gin.Context) {
		c.File("javanese-chess.log")
	})

	// WebSocket
	r.GET("/ws", hub.HandleWS)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
