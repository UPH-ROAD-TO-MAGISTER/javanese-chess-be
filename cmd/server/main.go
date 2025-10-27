package main

import (
	httpapi "javanese-chess/internal/api/http"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"javanese-chess/internal/store"
	"log"
	"os"
)

func main() {
	cfg := config.Load()
	mem := store.NewMemoryStore()
	rm := room.NewManager(mem, cfg)
	r := httpapi.NewRouter(rm, cfg)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
