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
// @Router /api/config/weights/default [get]
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
// @Router /api/config/weights/room [get]
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
