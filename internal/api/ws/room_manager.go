package ws

import "javanese-chess/internal/shared"

type RoomManager interface {
	Get(roomCode string) (*shared.Room, bool)
	ApplyMove(room *shared.Room, playerID string, x, y, card int) error
	BotMove(room *shared.Room, botID string) (shared.Move, error)
}
