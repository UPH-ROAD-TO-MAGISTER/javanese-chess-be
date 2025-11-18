package http

import (
	"net/http"

	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"

	"github.com/gin-gonic/gin"
)

// @Summary Add bots to a room or create room and apply config
// @Description Initialize room (create if missing), add bots and apply provided heuristic weights in one request
// @Tags Room
// @Accept json
// @Produce json
// @Param request body PlayRequest true "Room info"
// @Success 200 {object} map[string]interface{}
// @Router /api/play [post]
func PlayHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var playRequest PlayRequest
		if err := c.BindJSON(&playRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if playRequest.NumberBot < 0 {
			playRequest.NumberBot = 0
		}

		// Validate RoomID is provided
		if playRequest.RoomID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
			return
		}

		// Get existing room (must exist from room_created event)
		rx, ok := rm.Get(playRequest.RoomID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room not found"})
			return
		}

		// Validate room is in lobby state
		if rx.Status != "lobby" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "game has already started"})
			return
		}

		// Validate player names are provided
		if len(playRequest.PlayerName) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "player_name array is required"})
			return
		}

		// Add bots if requested
		if playRequest.NumberBot > 0 {
			rm.AddBots(rx, playRequest.NumberBot)
		}

		// Apply weights if provided
		if playRequest.Weights != nil {
			if !playRequest.Weights.ValidateWeights() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "weights must be non-negative"})
				return
			}
			if rx.RoomConfig == nil {
				rx.RoomConfig = config.NewRoomConfig(rx.Code)
			}
			rx.RoomConfig.SetWeights(*playRequest.Weights)
		}

		// Start the game (change status from lobby to playing)
		rm.StartGame(rx)

		// Broadcast game started to all clients
		hub.Broadcast(rx.Code, "game_started", gin.H{
			"room_code":  rx.Code,
			"turn_order": rx.TurnOrder,
			"players":    rx.Players,
			"board":      rx.Board,
			"status":     "playing",
		})

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"room_code":  rx.Code,
				"turn_order": rx.TurnOrder, // Shuffled player IDs
				"players":    rx.Players,   // Detailed player information
				"board":      rx.Board,
				"status":     "playing",
			},
		})
	}
}

// @Summary Join an existing room
// @Description Join an existing room with a room code
// @Tags Room
// @Accept json
// @Produce json
// @Param request body JoinRoomRequest true "Join room info"
// @Success 200 {object} map[string]interface{}
// @Router /api/join [post]
func JoinRoomHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var joinRequest JoinRoomRequest
		if err := c.BindJSON(&joinRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if joinRequest.RoomCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room_code is required"})
			return
		}

		if joinRequest.PlayerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "player_name is required"})
			return
		}

		// Validate room exists
		rx, ok := rm.Get(joinRequest.RoomCode)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room not found"})
			return
		}

		// Validate room is in lobby state
		if rx.Status != "lobby" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "game has already started"})
			return
		}

		// Join the room
		rx, err := rm.JoinRoom(joinRequest.RoomCode, joinRequest.PlayerName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Broadcast only the new player's name
		hub.Broadcast(rx.Code, "new_player_joined", gin.H{
			"player_name": joinRequest.PlayerName,
		})

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"room_code":  rx.Code,
				"turn_order": rx.TurnOrder,
				"players":    rx.Players,
				"board":      rx.Board,
				"status":     "playing",
			},
		})
	}
}
