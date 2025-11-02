package room

import (
	"errors"
	"javanese-chess/internal/api/ws"
	"javanese-chess/internal/config"
	"javanese-chess/internal/game"
	"javanese-chess/internal/shared"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Manager struct {
	store Store
	cfg   config.Config
	hub   *ws.Hub
}

func NewManager(s Store, cfg config.Config, hub *ws.Hub) *Manager {
	return &Manager{store: s, cfg: cfg, hub: hub}
}

func (m *Manager) CreateRoom(creatorName string) *shared.Room {
	code := randCode(6)
	r := &shared.Room{
		Code:       code,
		Board:      game.NewBoard(m.cfg.BoardSize),
		TurnIdx:    0,
		CreatedAt:  time.Now(),
		Cfg:        m.cfg,
		RoomConfig: config.NewRoomConfig(code),
		Players: []shared.Player{
			{
				ID:    uuid.NewString(),
				Name:  creatorName,
				IsBot: false,
				Hand:  []int{1, 2, 3},
			},
		},
	}
	game.UpdateVState(&r.Board)
	m.store.SaveRoom(r)
	return r
}

func NewRoomWithID(roomID, creatorName string) *shared.Room {
	if creatorName == "" {
		creatorName = "Player"
	}

	// Use the default configuration for the room
	defaultCfg := config.Get()

	// Create a new board with the default configuration
	board := game.NewBoard(defaultCfg.BoardSize)

	// Generate and shuffle the deck for the first player
	deck := GenerateDeck()

	// Draw the initial 3 cards
	initialHand := deck[:3]
	deck = deck[3:]

	r := &shared.Room{
		Code:       roomID, // Use the provided RoomID as the Code
		Board:      board,
		TurnIdx:    0,
		CreatedAt:  time.Now(),
		Cfg:        *defaultCfg,
		RoomConfig: config.NewRoomConfig(roomID),
		Players: []shared.Player{
			{
				ID:    uuid.NewString(),
				Name:  creatorName,
				IsBot: false,
				Hand:  initialHand,
				Deck:  deck,
			},
		},
	}

	// Update the board's virtual state
	game.UpdateVState(&r.Board)

	return r
}

// GenerateDeck creates a shuffled deck of 18 cards (two sets of 1-9)
func GenerateDeck() []int {
	deck := make([]int, 18)
	for i := 0; i < 9; i++ {
		deck[i] = i + 1
		deck[i+9] = i + 1
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func (m *Manager) CreateRoomWithID(roomID, playerName string) *shared.Room {
	room := NewRoomWithID(roomID, playerName)
	m.store.SaveRoom(room)
	return room
}

func (m *Manager) AddBots(r *shared.Room, n int) {
	// Ensure the human player is included in the shuffle
	if len(r.Players) == 0 {
		// Generate a unique deck for the human player
		deck := GenerateDeck()
		hand := deck[:3]
		deck = deck[3:]

		r.Players = append(r.Players, shared.Player{
			ID:    uuid.NewString(),
			Name:  "Human Player",
			IsBot: false,
			Hand:  hand,
			Deck:  deck,
		})
	}

	for i := 0; i < n; i++ {
		// Generate a unique deck for the bot
		deck := GenerateDeck()
		// Assign the first 3 cards to the bot's hand
		hand := deck[:3]
		deck = deck[3:]

		r.Players = append(r.Players, shared.Player{
			ID:    "bot-" + uuid.NewString(),
			Name:  "Bot",
			IsBot: true,
			Hand:  hand,
			Deck:  deck,
		})
	}

	// Shuffle the players
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(r.Players), func(i, j int) {
		r.Players[i], r.Players[j] = r.Players[j], r.Players[i]
	})

	// Update turn order based on shuffled players
	r.TurnOrder = make([]string, len(r.Players))
	for i, player := range r.Players {
		r.TurnOrder[i] = player.ID
	}

	m.store.SaveRoom(r)
}

func (m *Manager) Get(code string) (*shared.Room, bool) {
	return m.store.GetRoom(code)
}

func (m *Manager) currentPlayer(r *shared.Room) *shared.Player {
	if len(r.Players) == 0 {
		return nil
	}
	return &r.Players[r.TurnIdx%len(r.Players)]
}

func (m *Manager) ApplyMove(r *shared.Room, playerID string, x, y, card int) error {
	cp := m.currentPlayer(r)
	if cp == nil || cp.ID != playerID {
		return errors.New("not your turn or player invalid")
	}

	// Ensure the move is legal
	legal := false
	for _, mv := range game.GenerateLegalMoves(r.Board, cp.Hand, playerID) {
		if mv.X == x && mv.Y == y && mv.Card == card {
			legal = true
			break
		}
	}
	if !legal {
		return errors.New("illegal move")
	}

	// Apply the move to the board
	game.ApplyMove(&r.Board, x, y, playerID, card)

	// Remove the card from the player's hand
	for i, v := range cp.Hand {
		if v == card {
			cp.Hand = append(cp.Hand[:i], cp.Hand[i+1:]...)
			break
		}
	}
	game.UpdateVState(&r.Board)

	// Draw a new card from the deck
	if len(cp.Deck) > 0 {
		cp.Hand = append(cp.Hand, cp.Deck[0])
		cp.Deck = cp.Deck[1:]
	}

	// Check for a winning move
	if game.IsWinningAfter(r.Board, x, y, playerID, card) {
		r.WinnerID = &playerID
		m.hub.Broadcast(r.Code, "game_over", gin.H{
			"winner": playerID,
			"board":  r.Board,
		})
		return nil
	}

	// Update the turn index to the next player
	r.TurnIdx = (r.TurnIdx + 1) % len(r.Players)

	// Broadcast the updated game state
	m.hub.Broadcast(r.Code, "move", gin.H{
		"playerID": playerID,
		"x":        x,
		"y":        y,
		"card":     card,
		"board":    r.Board,
		"nextTurn": r.Players[r.TurnIdx].ID,
	})

	// Save the updated room state
	m.store.SaveRoom(r)
	return nil
}

func (m *Manager) BotMove(r *shared.Room, botID string) (shared.Move, error) {
	cp := m.currentPlayer(r)
	if cp == nil || cp.ID != botID {
		return shared.Move{}, errors.New("not bot's turn")
	}

	// Generate all legal moves for the bot
	cands := game.GenerateLegalMoves(r.Board, cp.Hand, botID)
	if len(cands) == 0 {
		return shared.Move{}, errors.New("no legal moves available")
	}

	// Get room-specific weights, or use defaults
	weights := r.Cfg.DefaultWeights
	if r.RoomConfig != nil {
		weights = r.RoomConfig.GetWeights()
	}

	// Find the best move based on heuristic scores
	best, err := m.findBestMove(r.Board, cands, cp.Hand, weights)
	if err != nil {
		return shared.Move{}, err
	}

	// Apply the best move
	if err := m.ApplyMove(r, botID, best.X, best.Y, best.Card); err != nil {
		return shared.Move{}, err
	}

	return shared.Move{
		X:        best.X,
		Y:        best.Y,
		Card:     best.Card,
		PlayerID: botID,
	}, nil
}

// Helper function to find the best move based on heuristic scores
func (m *Manager) findBestMove(board game.Board, cands []game.Move, hand []int, weights config.HeuristicWeights) (game.Move, error) {
	if len(cands) == 0 {
		return game.Move{}, errors.New("no candidates for best move")
	}

	best := cands[0]
	bestScore := game.HeuristicScore(board, best, hand, weights)
	for _, c := range cands[1:] {
		if score := game.HeuristicScore(board, c, hand, weights); score > bestScore {
			best = c
			bestScore = score
		}
	}

	return best, nil
}

func (m *Manager) CheckEndgame(r *shared.Room) {
	// Check if there is already a winner
	if r.WinnerID != nil {
		return
	}

	// Check if no moves are left for all players
	noMovesLeft := true
	for _, player := range r.Players {
		if len(game.GenerateLegalMoves(r.Board, player.Hand, player.ID)) > 0 {
			noMovesLeft = false
			break
		}
	}

	if noMovesLeft {
		// Determine the winner based on adjacent card values
		m.determineWinnerByAdjacentValues(r)
	}
}

func (m *Manager) determineWinnerByAdjacentValues(r *shared.Room) {
	playerScores := make(map[string]int)

	// Calculate scores for each player based on adjacent card values
	for x := 0; x < r.Board.Size; x++ {
		for y := 0; y < r.Board.Size; y++ {
			cell := r.Board.Cells[x][y]
			if cell.OwnerID != "" {
				playerScores[cell.OwnerID] += m.calculateAdjacentCardValue(r.Board, x, y)
			}
		}
	}

	// Find the player with the highest score
	var winnerID string
	highestScore := -1
	for playerID, score := range playerScores {
		if score > highestScore {
			highestScore = score
			winnerID = playerID
		}
	}

	// Set the winner
	if winnerID != "" {
		r.WinnerID = &winnerID
	}
}

func (m *Manager) calculateAdjacentCardValue(board game.Board, x, y int) int {
	totalValue := 0
	directions := []struct{ dx, dy int }{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1}, // Horizontal and vertical
		{-1, -1}, {1, 1}, {-1, 1}, {1, -1}, // Diagonal
	}

	for _, dir := range directions {
		nx, ny := x+dir.dx, y+dir.dy
		if nx >= 0 && ny >= 0 && nx < board.Size && ny < board.Size {
			totalValue += board.Cells[nx][ny].Value
		}
	}

	return totalValue
}

const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randCode(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

type RankRow struct {
	PlayerID string `json:"playerId"`
	LineSum  int    `json:"tieBreakerLineSum"`
	TotalSum int    `json:"totalCellsSum"`
}

func (m *Manager) Rank(r *shared.Room) []RankRow {
	out := make([]RankRow, 0, len(r.Players))
	for _, p := range r.Players {
		out = append(out, RankRow{
			PlayerID: p.ID,
			LineSum:  game.TieBreakerLineSum(r.Board, p.ID),
			TotalSum: game.TotalOwnedSum(r.Board, p.ID),
		})
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].LineSum > out[i].LineSum || (out[j].LineSum == out[i].LineSum && out[j].TotalSum > out[i].TotalSum) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
