package main

import (
	httpapi "javanese-chess/internal/api/http"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"javanese-chess/internal/store"
	"log"
	"net/http"
	"os"

	// swagger packages
	_ "javanese-chess/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Javanese Chess Bot API
// @version 1.0
// @description REST API for heuristic-based chess-like bot (Go + Gin)
// @contact.name Backend Team
// @contact.email backend@yourcompany.com
// @host localhost:8080
// @BasePath /
func main() {
	cfg := config.Load()
	mem := store.NewMemoryStore()
	rm := room.NewManager(mem, cfg)
	r := httpapi.NewRouter(rm, cfg)

	r.Use(cors.Default())

	// Alternatively, to specify allowed origins or other rules:
	// r.Use(cors.New(cors.Config{
	//     AllowOrigins:     []string{"https://example.com", "http://localhost:8080"},
	//     AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
	//     AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	//     AllowCredentials: true,
	// }))

	// Swagger route
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
