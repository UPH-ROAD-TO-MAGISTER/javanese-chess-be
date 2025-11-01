package http

import (
	"encoding/json"
	"fmt"
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"
	"math/rand"
	"net/http"
	"time"

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
		var rx *room.Room
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

		// Prepare response structure matching your example
		turnOrder := []string{}
		playersOut := []map[string]interface{}{}

		// assign colors from enum (red, green, blue, purple) shuffled across players/bots
		colors := []string{"red", "green", "blue", "purple"}
		totalPlayers := len(rx.Players)
		colorPool := make([]string, totalPlayers)
		for i := 0; i < totalPlayers; i++ {
			colorPool[i] = colors[i%len(colors)]
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if totalPlayers > 1 {
			r.Shuffle(len(colorPool), func(i, j int) { colorPool[i], colorPool[j] = colorPool[j], colorPool[i] })
		}

		for pi, p := range rx.Players {
			// marshal player to generic map to avoid compile-time field assumptions
			b, _ := json.Marshal(p)
			var pm map[string]interface{}
			_ = json.Unmarshal(b, &pm)

			// extract player id
			playerID := ""
			if v, ok := pm["player_id"].(string); ok {
				playerID = v
			} else if v, ok := pm["id"].(string); ok {
				playerID = v
			} else if v, ok := pm["ID"].(string); ok {
				playerID = v
			}
			if playerID == "" {
				// fallback: try struct field ID via fmt (best-effort)
				playerID = fmt.Sprintf("%v", pm["ID"])
			}
			if playerID == "" {
				// final fallback generate one
				playerID = fmt.Sprintf("P-%d", pi+1)
			}
			turnOrder = append(turnOrder, playerID)

			// assign shuffled color (override any existing color)
			color := ""
			if pi < len(colorPool) {
				color = colorPool[pi]
				pm["color"] = color
			} else if v, ok := pm["color"].(string); ok {
				color = v
			}

			name := ""
			if v, ok := pm["name"].(string); ok {
				name = v
			}

			isBot := false
			if v, ok := pm["is_bot"].(bool); ok {
				isBot = v
			} else if v, ok := pm["isBot"].(bool); ok {
				isBot = v
			}

			// normalize hand -> []{id, value, color}
			handOut := []map[string]interface{}{}
			if h, ok := pm["hand"].([]interface{}); ok {
				for ci, hv := range h {
					val := 0
					switch t := hv.(type) {
					case float64:
						val = int(t)
					case int:
						val = t
					case map[string]interface{}:
						if vv, ok := t["value"].(float64); ok {
							val = int(vv)
						}
					}
					cardID := fmt.Sprintf("card-%d", (pi*10)+(ci+1))
					handOut = append(handOut, map[string]interface{}{
						"id":    cardID,
						"value": val,
						"color": color,
					})
				}
			}

			deckCount := 0
			if v, ok := pm["deck_count"].(float64); ok {
				deckCount = int(v)
			} else if v, ok := pm["deckCount"].(float64); ok {
				deckCount = int(v)
			}

			playerMap := map[string]interface{}{
				"player_id":  playerID,
				"name":       name,
				"color":      color,
				"hand":       handOut,
				"deck_count": deckCount,
			}
			if isBot {
				playerMap["is_bot"] = true
			}
			playersOut = append(playersOut, playerMap)
		}

		// Randomize turnOrder and players order so player turn is random
		ids := make([]string, len(turnOrder))
		copy(ids, turnOrder)
		if len(ids) > 1 {
			r.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		}
		turnOrder = ids

		// board as-is (will marshal to JSON)
		boardOut := rx.Board

		// status extraction
		status := "playing"
		{
			b, _ := json.Marshal(rx)
			var rmMap map[string]interface{}
			_ = json.Unmarshal(b, &rmMap)
			if s, ok := rmMap["status"].(string); ok && s != "" {
				status = s
			}
		}

		// Notify clients
		hub.Broadcast(rx.Code, "state-updated", gin.H{"room": rx})

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"room_code":  rx.Code,
				"turn_order": turnOrder,
				"players":    playersOut,
				"board":      boardOut,
				"status":     status,
			},
		})
	}
}
