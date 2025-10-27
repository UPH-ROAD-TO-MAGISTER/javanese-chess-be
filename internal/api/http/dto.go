package http

// CreateRoomRequest represents the payload for /create-room.
type CreateRoomRequest struct {
	PlayerName string `json:"playerName"`
}

// PlayRequest represents the payload for /play.
type PlayRequest struct {
	RoomCode string `json:"roomCode"`
	Bots     int    `json:"bots"`
}

// MoveRequest represents a player move.
type MoveRequest struct {
	RoomCode string `json:"roomCode"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Value    int    `json:"value"`
	PlayerID string `json:"playerId"`
}

// MoveBotRequest represents a bot move.
type MoveBotRequest struct {
	RoomCode string `json:"roomCode"`
	BotID    string `json:"botId"`
	Hold     []int  `json:"holdCards"`
}

// SetHandsRequest represents FE-shuffled cards.
type SetHandsRequest struct {
	RoomCode string       `json:"roomCode"`
	Hands    []PlayerHand `json:"hands"`
}

type PlayerHand struct {
	PlayerID string `json:"playerId"`
	Cards    []int  `json:"cards"`
}
