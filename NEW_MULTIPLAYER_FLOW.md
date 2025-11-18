# New Multiplayer Flow Implementation

## Overview
The game flow has been updated to support a lobby system where players can join a room before the game starts.

## Flow Summary

### 1. Room Creation (WebSocket)
**Frontend → Backend**
- Action: `room_created`
- Room code: Provided by FE
- Backend stores the room in "lobby" state

**Backend → Frontend**
- Action: `room_created`
- Data: `{ room_code, status: "lobby" }`

### 2. Player Joining (HTTP API)
**Frontend → Backend**
- Endpoint: `POST /api/join`
- Body: `{ room_code, player_name }`
- Backend validates room exists and is in lobby state

**Backend → Frontend (WebSocket Broadcast)**
- Action: `new_player_joined`
- Data: `{ player_name }`
- Sent to all clients in the room

### 3. Game Start (HTTP API)
**Frontend → Backend**
- Endpoint: `POST /api/play`
- Body: `{ room_id, player_name: [], number_bot, number_player, weights? }`
- Note: `player_name` is now an **array of strings**

**Backend → Frontend (WebSocket Broadcast)**
- Action: `game_started`
- Data: `{ room_code, turn_order, players, board, status: "playing" }`
- Sent to all clients in the room

## Key Changes

### 1. Room Status Field
- Added `Status` field to `Room` struct
- Values: `"lobby"` or `"playing"`
- Lobby: Room created but game not started
- Playing: Game in progress

### 2. WebSocket Handler
**New Action**: `room_created`
```go
case "room_created":
    h.handleRoomCreated(roomCode, msg.Data)
```

### 3. API Changes

#### POST /api/join
- **Validates**: Room exists AND room is in lobby state
- **Broadcasts**: `new_player_joined` with player name only
- **Returns**: Room data with lobby status

#### POST /api/play
- **Requires**: `room_id` (must exist from `room_created`)
- **Changed**: `player_name` from `string` to `[]string`
- **Validates**: Room exists AND room is in lobby state
- **Action**: Transitions room from "lobby" to "playing"
- **Broadcasts**: `game_started` to all clients

### 4. Room Manager Methods

#### CreateLobbyRoom(roomCode string)
- Creates room in lobby state
- No players initially
- Board initialized with center cell VState=1

#### StartGame(room *shared.Room)
- Changes room status from "lobby" to "playing"
- Called by `/api/play` endpoint

## Complete Flow Example

```
1. FE: WebSocket → room_created { room_code: "ABC123" }
   BE: Creates lobby room
   BE: Broadcast → room_created { room_code: "ABC123", status: "lobby" }

2. FE: POST /api/join { room_code: "ABC123", player_name: "Alice" }
   BE: Adds Alice to room
   BE: Broadcast → new_player_joined { player_name: "Alice" }

3. FE: POST /api/join { room_code: "ABC123", player_name: "Bob" }
   BE: Adds Bob to room
   BE: Broadcast → new_player_joined { player_name: "Bob" }

4. FE: POST /api/play { 
       room_id: "ABC123", 
       player_name: ["Alice", "Bob"],
       number_bot: 2,
       number_player: 2
   }
   BE: Adds 2 bots, starts game
   BE: Broadcast → game_started { 
       room_code: "ABC123",
       turn_order: [...shuffled...],
       players: [Alice, Bob, Bot1, Bot2],
       board: {...},
       status: "playing"
   }

5. Game begins, human_move and bot_move actions work as before
```

## Backward Compatibility

The old flow still works:
- `CreateRoom()` - Creates room with status "playing"
- `CreateRoomWithID()` - Creates room with status "playing"

New flow uses:
- `CreateLobbyRoom()` - Creates room with status "lobby"
- `StartGame()` - Transitions to "playing"

## Testing Notes

1. Test room creation via WebSocket
2. Test multiple players joining before game starts
3. Test joining after game starts (should fail)
4. Test starting game with player_name array
5. Test game start broadcast to all clients
6. Verify bot turns still work after game starts
