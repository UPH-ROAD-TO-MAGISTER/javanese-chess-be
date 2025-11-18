package ws

import "javanese-chess/internal/shared"

type RoomManager interface {
	Get(roomCode string) (*shared.Room, bool)
	ApplyMove(room *shared.Room, playerID string, x, y, card int) error
	BotMove(room *shared.Room, botID string) (shared.Move, error)
	CreateLobbyRoom(roomCode string, roomMasterName string) *shared.Room
	JoinRoom(roomCode string, playerName string) (*shared.Room, error)
	StartGame(room *shared.Room)
}
