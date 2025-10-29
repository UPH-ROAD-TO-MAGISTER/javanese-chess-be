package http

import (
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/game"
	"javanese-chess/internal/room"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Create new room
// @Description Create a new room with a single human player
// @Tags Room
// @Accept json
// @Produce json
// @Param request body http.CreateRoomRequest true "Player info"
// @Success 200 {object} map[string]interface{}
// @Router /create-room [post]
func CreateRoomHandler(rm *room.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			PlayerName string `json:"playerName"`
		}
		if err := c.BindJSON(&req); err != nil || req.PlayerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "playerName required"})
			return
		}
		room := rm.CreateRoom(req.PlayerName)
		c.JSON(http.StatusOK, gin.H{"roomCode": room.Code, "room": room})
	}
}

// @Summary Add bots to a room
// @Description Initialize bots to start the game (default 1 bot if not specified)
// @Tags Room
// @Accept json
// @Produce json
// @Param request body PlayRequest true "Room info"
// @Success 200 {object} map[string]interface{}
// @Router /play [post]
func PlayHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// @Summary Get possible move boxes for player
// @Description Returns all available cells for a player's move (place/replace)
// @Tags Game
// @Produce json
// @Param roomCode query string true "Room Code"
// @Param playerId query string true "Player ID"
// @Success 200 {object} map[string]interface{}
// @Router /possible-moves [get]
func PossibleMovesHandler(rm *room.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// @Summary Set player hands (cards) manually
// @Description Apply shuffled hands from FE to room
// @Tags Game
// @Accept json
// @Produce json
// @Param request body SetHandsRequest true "Hands data"
// @Success 200 {object} map[string]interface{}
// @Router /set-hands [post]
func SetHandsHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// @Summary Player makes a move
// @Description Submit coordinates (x, y) and card value for player's move
// @Tags Game
// @Accept json
// @Produce json
// @Param request body MoveRequest true "Move data"
// @Success 200 {object} map[string]interface{}
// @Router /move [post]
func MoveHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// @Summary Let bot make its move
// @Description Bot picks the best move using heuristic evaluation
// @Tags Game
// @Accept json
// @Produce json
// @Param request body MoveBotRequest true "Bot move"
// @Success 200 {object} map[string]interface{}
// @Router /move-bot [post]
func MoveBotHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
