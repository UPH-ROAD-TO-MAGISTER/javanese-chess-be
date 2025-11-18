package http

import "javanese-chess/internal/config"

// CreateRoomRequest represents the payload for /create-room.
type CreateRoomRequest struct {
	PlayerName string `json:"player_name"`
}

// JoinRoomRequest represents the payload for joining an existing room.
type JoinRoomRequest struct {
	RoomCode   string `json:"room_code"`
	PlayerName string `json:"player_name"`
}

// PlayRequest represents the payload for /play.
type PlayRequest struct {
	NumberPlayer int                      `json:"number_player"`
	NumberBot    int                      `json:"number_bot"`
	RoomID       string                   `json:"room_id"`
	PlayerName   []string                 `json:"player_name"` // Changed to array
	Weights      *config.HeuristicWeights `json:"weights"`
}

// MoveRequest represents a player move.
type MoveRequest struct {
	RoomCode string `json:"room_code"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Value    int    `json:"value"`
	PlayerID string `json:"player_id"`
}

// MoveBotRequest represents a bot move.
type MoveBotRequest struct {
	RoomCode string `json:"room_code"`
	BotID    string `json:"bot_id"`
	Hold     []int  `json:"hold_cards"`
}

// SetHandsRequest represents FE-shuffled cards.
type SetHandsRequest struct {
	RoomCode string       `json:"room_code"`
	Hands    []PlayerHand `json:"hands"`
}

type PlayerHand struct {
	PlayerID string `json:"player_id"`
	Cards    []int  `json:"cards"`
}
