package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[*websocket.Conn]struct{}
	roomManager RoomManager
}

func NewHub(roomManager RoomManager) *Hub {
	log.Printf("Initializing Hub with RoomManager: %+v", roomManager)
	return &Hub{
		rooms:       make(map[string]map[*websocket.Conn]struct{}),
		roomManager: roomManager,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func (h *Hub) HandleWS(c *gin.Context) {
	log.Printf("HandleWS called. Hub state: %+v", h)

	roomCode := c.Query("room_code")
	// Room code is now optional - it can be provided later via room_created action

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	log.Printf("WebSocket connection established, initial room: %s", roomCode)

	// Add the connection to the room if room_code was provided
	if roomCode != "" {
		h.mu.Lock()
		if _, ok := h.rooms[roomCode]; !ok {
			h.rooms[roomCode] = make(map[*websocket.Conn]struct{})
		}
		h.rooms[roomCode][conn] = struct{}{}
		h.mu.Unlock()
	}

	// Track current room for this connection
	currentRoom := roomCode

	defer func() {
		h.mu.Lock()
		if currentRoom != "" {
			delete(h.rooms[currentRoom], conn)
		}
		h.mu.Unlock()
		_ = conn.Close()
	}()

	for {
		var msg struct {
			Action string      `json:"action"`
			Data   interface{} `json:"data"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			break
		}

		// Process the action
		switch msg.Action {
		case "room_created":
			// Extract room code from data
			newRoomCode := h.handleRoomCreated(conn, &currentRoom, msg.Data)
			if newRoomCode != "" {
				currentRoom = newRoomCode
			}
		case "human_move":
			h.handleHumanMove(currentRoom, msg.Data)
		case "bot_move":
			// Trigger bot move explicitly if requested (optional feature)
			room, ok := h.roomManager.Get(currentRoom)
			if !ok {
				log.Printf("Room not found: %s", currentRoom)
				continue
			}
			currentPlayer := room.Players[room.TurnIdx]
			if currentPlayer.IsBot {
				if botMove, err := h.roomManager.BotMove(room, currentPlayer.ID); err == nil {
					h.Broadcast(currentRoom, "bot_move", gin.H{
						"bot_id": currentPlayer.ID,
						"x":      botMove.X,
						"y":      botMove.Y,
						"card":   botMove.Card,
						"board":  room.Board,
					})
				} else {
					log.Printf("Failed to process bot move: %v", err)
				}
			}
		default:
			log.Printf("Unknown action: %s", msg.Action)
		}
	}
}

func (h *Hub) Broadcast(roomCode string, action string, data interface{}) {
	if h == nil {
		log.Printf("Hub instance is nil")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomCode]
	if !ok {
		return
	}

	message := map[string]interface{}{
		"action": action,
		"data":   data,
	}
	for conn := range clients {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Failed to send message: %v", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

func (h *Hub) handleHumanMove(roomCode string, data interface{}) {
	// Parse the move data
	var move struct {
		PlayerID string `json:"player_id"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
		Card     int    `json:"card"`
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		log.Printf("ERROR: Failed to marshal move data: %v", err)
		return
	}

	if err := json.Unmarshal(rawData, &move); err != nil {
		log.Printf("ERROR: Invalid move data: %v", err)
		return
	}

	log.Printf("=== WEBSOCKET HUMAN MOVE ===")
	log.Printf("Room: %s, PlayerID: %s, Position: (%d,%d), Card: %d", roomCode, move.PlayerID, move.X, move.Y, move.Card)

	// Get the room
	room, ok := h.roomManager.Get(roomCode)
	if !ok {
		log.Printf("ERROR: Room not found: %s", roomCode)
		h.Broadcast(roomCode, "error", map[string]interface{}{
			"message": "Room not found",
		})
		return
	}

	// Log board state for debugging
	boardEmpty := true
	placedCount := 0
	for y := 0; y < room.Board.Size; y++ {
		for x := 0; x < room.Board.Size; x++ {
			if room.Board.Cells[y][x].Value != 0 {
				boardEmpty = false
				placedCount++
				log.Printf("DEBUG: Card found at (%d,%d): value=%d, owner=%s",
					x, y, room.Board.Cells[y][x].Value, room.Board.Cells[y][x].OwnerID)
			}
		}
	}
	log.Printf("DEBUG: Board size=%d, isEmpty=%v, placedCards=%d", room.Board.Size, boardEmpty, placedCount)
	log.Printf("DEBUG: Center position should be: (%d,%d)", room.Board.Size/2, room.Board.Size/2)
	log.Printf("DEBUG: Received position: (%d,%d)", move.X, move.Y) // Apply the human move
	if err := h.roomManager.ApplyMove(room, move.PlayerID, move.X, move.Y, move.Card); err != nil {
		log.Printf("ERROR: Failed to apply move: %v", err)
		h.Broadcast(roomCode, "error", map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	log.Printf("SUCCESS: Move applied successfully")
	log.Printf("============================")

	// Broadcast the updated game state
	h.Broadcast(roomCode, "move", map[string]interface{}{
		"player_id": move.PlayerID,
		"x":         move.X,
		"y":         move.Y,
		"card":      move.Card,
		"board":     room.Board,
		"next_turn": room.Players[room.TurnIdx].ID,
	})

	// If it's the bot's turn, trigger the bot's move
	currentPlayer := room.Players[room.TurnIdx]
	if currentPlayer.IsBot {
		go func() {
			h.handleBotMove(roomCode)
		}()
	}
}

func (h *Hub) handleRoomCreated(conn *websocket.Conn, currentRoom *string, data interface{}) string {
	// Extract room code and player name from data
	var roomData struct {
		RoomCode   string `json:"room_code"`
		PlayerName string `json:"player_name"`
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		log.Printf("ERROR: Failed to marshal room data: %v", err)
		conn.WriteJSON(map[string]interface{}{
			"action": "error",
			"data":   map[string]interface{}{"message": "Invalid room data"},
		})
		return ""
	}

	if err := json.Unmarshal(rawData, &roomData); err != nil {
		log.Printf("ERROR: Invalid room data: %v", err)
		conn.WriteJSON(map[string]interface{}{
			"action": "error",
			"data":   map[string]interface{}{"message": "Invalid room data format"},
		})
		return ""
	}

	roomCode := roomData.RoomCode
	if roomCode == "" {
		log.Printf("ERROR: Room code not provided in data")
		conn.WriteJSON(map[string]interface{}{
			"action": "error",
			"data":   map[string]interface{}{"message": "room_code is required"},
		})
		return ""
	}

	playerName := roomData.PlayerName
	if playerName == "" {
		log.Printf("ERROR: Player name not provided in data")
		conn.WriteJSON(map[string]interface{}{
			"action": "error",
			"data":   map[string]interface{}{"message": "player_name is required"},
		})
		return ""
	}

	log.Printf("=== ROOM CREATED VIA WEBSOCKET ===")
	log.Printf("Room Code: %s, Room Master: %s", roomCode, playerName)

	// Create lobby room with room master as first player
	room := h.roomManager.CreateLobbyRoom(roomCode, playerName)
	if room == nil {
		log.Printf("ERROR: Failed to create lobby room")
		h.Broadcast(roomCode, "error", map[string]interface{}{
			"message": "Failed to create room",
		})
		return ""
	}

	// Add this connection to the room
	h.mu.Lock()
	if _, ok := h.rooms[roomCode]; !ok {
		h.rooms[roomCode] = make(map[*websocket.Conn]struct{})
	}
	h.rooms[roomCode][conn] = struct{}{}

	// Remove from old room if it existed
	if *currentRoom != "" && *currentRoom != roomCode {
		delete(h.rooms[*currentRoom], conn)
	}
	h.mu.Unlock()

	// Broadcast room created confirmation
	h.Broadcast(roomCode, "room_created", map[string]interface{}{
		"room_code": roomCode,
		"status":    "lobby",
	})

	log.Printf("SUCCESS: Lobby room created with code: %s", roomCode)
	log.Printf("===================================")

	return roomCode
}

func (h *Hub) handleBotMove(roomCode string) {
	// Keep processing bot moves while the current player is a bot
	for {
		// Get the room
		room, ok := h.roomManager.Get(roomCode)
		if !ok {
			log.Printf("Room not found: %s", roomCode)
			return
		}

		// Check if game is over
		if room.WinnerID != nil {
			log.Printf("Game is over, winner: %s", *room.WinnerID)
			return
		}

		// Get the current player
		currentPlayer := room.Players[room.TurnIdx]
		if !currentPlayer.IsBot {
			// Current player is human, stop the bot loop
			log.Printf("Current player is not a bot: %s", currentPlayer.ID)
			return
		}

		// Trigger the bot's move
		botMove, err := h.roomManager.BotMove(room, currentPlayer.ID)
		if err != nil {
			log.Printf("Failed to process bot move: %v", err)
			return
		}

		// Broadcast the bot's move
		h.Broadcast(roomCode, "bot_move", map[string]interface{}{
			"bot_id":    currentPlayer.ID,
			"x":         botMove.X,
			"y":         botMove.Y,
			"card":      botMove.Card,
			"board":     room.Board,
			"next_turn": room.Players[room.TurnIdx].ID,
		})

		// Check again if game is over after this bot move
		if room.WinnerID != nil {
			log.Printf("Game is over after bot move, winner: %s", *room.WinnerID)
			return
		}

		// Continue the loop - if next player is also a bot, it will process automatically
	}
}
