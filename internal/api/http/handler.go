package http

import (
	"net/http"

	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"javanese-chess/internal/shared"

	"github.com/gin-gonic/gin"
)

// @Summary Add bots to a room or create room and apply config
// @Description Initialize room (create if missing), add bots and apply provided heuristic weights in one request
// @Tags Room
// @Accept json
// @Produce json
// @Param request body PlayRequest true "Room info"
// @Success 200 {object} map[string]interface{}
// @Router /play [post]
func PlayHandler(rm *room.Manager, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var playRequest PlayRequest
		if err := c.BindJSON(&playRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if playRequest.NumberBot <= 0 {
			playRequest.NumberBot = 1
		}

		// Try to get an existing room by the provided RoomID
		var rx *shared.Room
		if playRequest.RoomID != "" {
			if r, ok := rm.Get(playRequest.RoomID); ok {
				rx = r
			} else {
				// If the room doesn't exist, create a new one with the given RoomID
				rx = rm.CreateRoomWithID(playRequest.RoomID, playRequest.PlayerName)
			}
		}

		// If room not found, create one (player name optional)
		if rx == nil {
			if playRequest.PlayerName == "" {
				playRequest.PlayerName = "Player"
			}
			rx = rm.CreateRoom(playRequest.PlayerName)
		}

		// Add bots
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

		// Notify clients of the initial game state
		hub.Broadcast(rx.Code, "state-updated", gin.H{"room": rx})

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
