package main

import (
	httpapi "javanese-chess/internal/api/http"
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"javanese-chess/internal/store"
	"log"
	"net/http"

	// swagger packages
	_ "javanese-chess/docs"

	"github.com/gin-gonic/gin"
)

// @title Javanese Chess Bot API
// @version 1.0
// @description REST API for heuristic-based chess-like bot (Go + Gin)
// @contact.name Backend Team
// @contact.email backend@yourcompany.com
// @BasePath /
func main() {
	cfg := config.Load()
	mem := store.NewMemoryStore()
	rm := room.NewManager(mem, *cfg)
	hub := ws.NewHub()
	r := httpapi.SetupRouter(rm, mem, hub)

	// Optional: Add root redirect to swagger
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// Use HTTP address from config (which reads from env or uses default)
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
