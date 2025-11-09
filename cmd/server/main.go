package main

import (
	"io"
	httpapi "javanese-chess/internal/api/http"
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"javanese-chess/internal/store"
	"log"
	"net/http"
	"os"

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
	// Setup logging to both file and console
	logFile, err := os.OpenFile("javanese-chess.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Warning: Could not open log file: %v. Logging to console only.", err)
	} else {
		defer logFile.Close()
		// Log to both file and console
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
		log.Println("=== Javanese Chess Server Started ===")
	}

	cfg := config.Load()
	mem := store.NewMemoryStore()
	hub := ws.NewHub(room.NewManager(mem, *cfg, nil))
	rm := room.NewManager(mem, *cfg, hub)

	// Create the Manager first, with a nil Hub
	rm = room.NewManager(mem, *cfg, nil)

	// Create the Hub, passing the Manager
	hub = ws.NewHub(rm)

	// Set the Hub in the Manager
	rm.SetHub(hub)

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
