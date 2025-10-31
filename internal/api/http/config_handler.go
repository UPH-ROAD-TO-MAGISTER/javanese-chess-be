package http

import (
	"net/http"

	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/room"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	store room.Store
	hub   *ws.Hub
}

func NewConfigHandler(s room.Store, hub *ws.Hub) *ConfigHandler {
	return &ConfigHandler{
		store: s,
		hub:   hub,
	}
}

// GetDefaultWeightsHandler returns the global default weights
// @Summary Get default heuristic weights
// @Description Returns the default heuristic weights based on research paper (Section 2.4)
// @Tags Config
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /config/weights/default [get]
func (h *ConfigHandler) GetDefaultWeightsHandler(c *gin.Context) {
	weights := config.Get().DefaultWeights

	c.JSON(http.StatusOK, gin.H{
		"weights": weights,
	})
}

// GetRoomWeightsHandler returns the weights for a specific room
// @Summary Get room heuristic weights
// @Description Returns the heuristic weights configured for a specific room
// @Tags Config
// @Produce json
// @Param roomCode query string true "Room Code"
// @Success 200 {object} map[string]interface{}
// @Router /config/weights/room [get]
func (h *ConfigHandler) GetRoomWeightsHandler(c *gin.Context) {
	roomCode := c.Query("roomCode")
	if roomCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomCode is required"})
		return
	}

	rm, ok := h.store.GetRoom(roomCode)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	// Get weights from RoomConfig or use defaults
	var weights config.HeuristicWeights
	var isCustomized bool

	if rm.RoomConfig != nil {
		weights = rm.RoomConfig.GetWeights()
		isCustomized = rm.RoomConfig.IsCustomized()
	} else {
		weights = config.Get().DefaultWeights
		isCustomized = false
	}

	c.JSON(http.StatusOK, gin.H{
		"room_code":     roomCode,
		"weights":       weights,
		"is_customized": isCustomized,
	})
}

type UpdateRoomWeightsRequest struct {
	RoomCode string                  `json:"room_code" binding:"required"`
	Weights  config.HeuristicWeights `json:"weights" binding:"required"`
}

// UpdateRoomWeightsHandler updates weights for a specific room
// @Summary Update room heuristic weights
// @Description Updates the heuristic weights for all bots in a specific room. Weights must be non-negative.
// @Tags Config
// @Accept json
// @Produce json
// @Param request body UpdateRoomWeightsRequest true "Update Request"
// @Success 200 {object} map[string]interface{}
// @Router /config/weights/room [post]
func (h *ConfigHandler) UpdateRoomWeightsHandler(c *gin.Context) {
	var req UpdateRoomWeightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate weights
	if !req.Weights.ValidateWeights() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "all weights must be non-negative integers",
		})
		return
	}

	rm, ok := h.store.GetRoom(req.RoomCode)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	// Initialize RoomConfig if it doesn't exist
	if rm.RoomConfig == nil {
		rm.RoomConfig = config.NewRoomConfig(req.RoomCode)
	}

	// Update room weights
	rm.RoomConfig.SetWeights(req.Weights)

	// Save room with updated config
	h.store.SaveRoom(rm)

	// Broadcast config update to all clients in the room
	h.hub.Broadcast(req.RoomCode, "room-config-updated", gin.H{
		"room_code": req.RoomCode,
		"weights":   req.Weights,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "room weights updated successfully",
		"room_code": req.RoomCode,
		"weights":   req.Weights,
	})
}

// ResetRoomWeightsHandler resets a room's weights to default
// @Summary Reset room weights to default
// @Description Resets a room's heuristic weights to the global defaults from research paper
// @Tags Config
// @Accept json
// @Produce json
// @Param roomCode query string true "Room Code"
// @Success 200 {object} map[string]interface{}
// @Router /config/weights/room/reset [post]
func (h *ConfigHandler) ResetRoomWeightsHandler(c *gin.Context) {
	roomCode := c.Query("roomCode")
	if roomCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomCode is required"})
		return
	}

	rm, ok := h.store.GetRoom(roomCode)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	// Reset to default weights
	defaultWeights := config.Get().DefaultWeights

	// Initialize RoomConfig if it doesn't exist
	if rm.RoomConfig == nil {
		rm.RoomConfig = config.NewRoomConfig(roomCode)
	}

	rm.RoomConfig.SetWeights(defaultWeights)

	h.store.SaveRoom(rm)

	// Broadcast reset to all clients
	h.hub.Broadcast(roomCode, "room-config-updated", gin.H{
		"room_code": roomCode,
		"weights":   defaultWeights,
		"reset":     true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "room weights reset to default (research paper values)",
		"weights": defaultWeights,
	})
}
