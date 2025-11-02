package room

type Broadcaster interface {
	Broadcast(roomCode string, action string, data interface{})
}
