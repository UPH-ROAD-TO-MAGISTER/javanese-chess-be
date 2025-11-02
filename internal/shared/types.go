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
	TurnIdx    int                `json:"turnIdx"`
	WinnerID   *string            `json:"winnerId"`
	Draw       bool               `json:"draw"`
	CreatedAt  time.Time          `json:"createdAt"`
	Cfg        config.Config      `json:"-"`
	RoomConfig *config.RoomConfig `json:"roomConfig,omitempty"`
	TurnOrder  []string           `json:"turnOrder"`
}

type Move struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Card     int    `json:"card"`
	PlayerID string `json:"playerId"`
}

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	IsBot bool   `json:"isBot"`
	Hand  []int  `json:"hand"`
	Deck  []int  `json:"deck"`
}
