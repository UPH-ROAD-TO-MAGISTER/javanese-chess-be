package shared

import (
	"javanese-chess/internal/config"
	"javanese-chess/internal/game"
	"time"
)

type Room struct {
	Code       string             `json:"code"`
	Board      game.Board         `json:"board"`
	Players    []Player           `json:"players"`
	TurnIdx    int                `json:"turn_idx"`
	WinnerID   *string            `json:"winner_id"`
	Draw       bool               `json:"draw"`
	CreatedAt  time.Time          `json:"created_at"`
	Cfg        config.Config      `json:"-"`
	RoomConfig *config.RoomConfig `json:"room_config,omitempty"`
	TurnOrder  []string           `json:"turn_order"`
	Status     string             `json:"status"` // "lobby" or "playing"
}

type Move struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Card     int    `json:"card"`
	PlayerID string `json:"player_id"`
}

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	IsBot bool   `json:"isBot"`
	Hand  []int  `json:"hand"`
	Deck  []int  `json:"-"`
	Color string `json:"color"` // Added field for player color
}
