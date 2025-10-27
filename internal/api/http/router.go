package http

import (
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/game"
	"javanese-chess/internal/room"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(rm *room.Manager, cfg config.Config) *gin.Engine {
	r := gin.Default()
	hub := ws.NewHub()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "heuristic-bot is running")
	})

	// WebSocket for FE live updates
	r.GET("/ws", hub.HandleWS)

	// Room/create
	r.POST("/create-room", func(c *gin.Context) {
		var req struct {
			PlayerName string `json:"playerName"`
		}
		if err := c.BindJSON(&req); err != nil || req.PlayerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "playerName required"})
			return
		}
		room := rm.CreateRoom(req.PlayerName)
		c.JSON(http.StatusOK, gin.H{"roomCode": room.Code, "room": room})
	})

	// Add bots (2-player scenario: 1 human + 1 bot)
	r.POST("/play", func(c *gin.Context) {
		var req struct {
			RoomCode string `json:"roomCode"`
			Bots     int    `json:"bots"`
		}
		if err := c.BindJSON(&req); err != nil || req.RoomCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "roomCode required"})
			return
		}
		rx, ok := rm.Get(req.RoomCode)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		if req.Bots <= 0 {
			req.Bots = 1
		}
		rm.AddBots(rx, req.Bots)
		hub.Broadcast(req.RoomCode, "state-updated", gin.H{"room": rx})
		c.JSON(http.StatusOK, gin.H{"room": rx})
	})

	// FE highlight: unique boxes (place/replace) without card permutations
	r.GET("/possible-moves", func(c *gin.Context) {
		roomCode := c.Query("roomCode")
		playerID := c.Query("playerId")
		rx, ok := rm.Get(roomCode)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		var player *room.Player
		for i := range rx.Players {
			if rx.Players[i].ID == playerID {
				player = &rx.Players[i]
				break
			}
		}
		if player == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "player not found"})
			return
		}
		moves := game.GenerateLegalMoves(rx.Board, player.Hand, playerID)
		type Box struct {
			X, Y int
			Mode string
		}
		seen := map[[2]int]string{}
		for _, m := range moves {
			mode := "place"
			cell := rx.Board.Cells[m.Y][m.X]
			if cell.OwnerID != "" && cell.OwnerID != playerID {
				mode = "replace"
			}
			key := [2]int{m.X, m.Y}
			if _, ok := seen[key]; !ok {
				seen[key] = mode
			} else if seen[key] != "replace" && mode == "replace" {
				seen[key] = mode
			}
		}
		out := []Box{}
		for k, v := range seen {
			out = append(out, Box{X: k[0], Y: k[1], Mode: v})
		}
		c.JSON(http.StatusOK, gin.H{"boxes": out})
	})

	// FE-shuffled hands setter
	r.POST("/set-hands", func(c *gin.Context) {
		var req struct {
			RoomCode string `json:"roomCode"`
			Hands    []struct {
				PlayerID string `json:"playerId"`
				Cards    []int  `json:"cards"`
			} `json:"hands"`
		}
		if err := c.BindJSON(&req); err != nil || req.RoomCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		rx, ok := rm.Get(req.RoomCode)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		for _, h := range req.Hands {
			for i := range rx.Players {
				if rx.Players[i].ID == h.PlayerID {
					rx.Players[i].Hand = append([]int(nil), h.Cards...)
					break
				}
			}
		}
		hub.Broadcast(req.RoomCode, "state-updated", gin.H{"room": rx})
		c.JSON(http.StatusOK, gin.H{"room": rx})
	})

	// Heuristic weights runtime config
	r.GET("/config/weights", func(c *gin.Context) {
		c.JSON(http.StatusOK, cfg.Weights)
	})
	r.POST("/config/weights", func(c *gin.Context) {
		var w config.Weights
		if err := c.BindJSON(&w); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid weights"})
			return
		}
		cfg.Weights = w
		hub.Broadcast("ALL", "config-updated", w)
		c.JSON(http.StatusOK, gin.H{"ok": true, "weights": w})
	})

	// Human move
	r.POST("/move", func(c *gin.Context) {
		var req struct {
			RoomCode string `json:"roomCode"`
			X        int    `json:"x"`
			Y        int    `json:"y"`
			Value    int    `json:"value"`
			PlayerID string `json:"playerId"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		rx, ok := rm.Get(req.RoomCode)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		if err := rm.ApplyMove(rx, req.PlayerID, req.X, req.Y, req.Value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hub.Broadcast(req.RoomCode, "move-applied", gin.H{"room": rx})
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"room":   rx,
			"winner": rx.WinnerID,
			"draw":   rx.Draw,
			"rank":   rm.Rank(rx),
		})
	})

	// Bot move
	r.POST("/move-bot", func(c *gin.Context) {
		var req struct {
			RoomCode string `json:"roomCode"`
			BotID    string `json:"botId"`
			Hold     []int  `json:"holdCards"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		rx, ok := rm.Get(req.RoomCode)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		for i := range rx.Players {
			if rx.Players[i].ID == req.BotID {
				if len(req.Hold) > 0 {
					rx.Players[i].Hand = append([]int(nil), req.Hold...)
				}
				break
			}
		}
		mv, err := rm.BotMove(rx, req.BotID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hub.Broadcast(req.RoomCode, "move-applied", gin.H{"room": rx})
		c.JSON(http.StatusOK, gin.H{
			"x": mv.X, "y": mv.Y, "value": mv.Card,
			"lastState": gin.H{"winner": rx.WinnerID, "draw": rx.Draw},
			"rank":      rm.Rank(rx),
			"room":      rx,
		})
	})

	return r
}
