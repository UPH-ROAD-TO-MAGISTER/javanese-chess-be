package room

import (
	"errors"
	"javanese-chess/internal/config"
	"javanese-chess/internal/game"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	IsBot bool   `json:"isBot"`
	Hand  []int  `json:"hand"`
	Index int    `json:"index"`
}

type Room struct {
	ID        string        `json:"id"`
	Code      string        `json:"code"`
	Board     game.Board    `json:"board"`
	Players   []Player      `json:"players"`
	TurnIdx   int           `json:"turnIdx"`
	WinnerID  *string       `json:"winnerId,omitempty"`
	Draw      bool          `json:"draw"`
	CreatedAt time.Time     `json:"createdAt"`
	Cfg       config.Config `json:"-"`
}

type Store interface {
	GetRoom(code string) (*Room, bool)
	SaveRoom(r *Room)
}

type Manager struct {
	store Store
	cfg   config.Config
}

func NewManager(s Store, cfg config.Config) *Manager {
	return &Manager{store: s, cfg: cfg}
}

func (m *Manager) CreateRoom(creatorName string) *Room {
	code := randCode(6)
	r := &Room{
		ID:        uuid.NewString(),
		Code:      code,
		Board:     game.NewBoard(m.cfg.Weights.BoardSize),
		TurnIdx:   0,
		CreatedAt: time.Now(),
		Cfg:       m.cfg,
	}
	game.UpdateVState(&r.Board)
	r.Players = append(r.Players, Player{
		ID:    uuid.NewString(),
		Name:  creatorName,
		IsBot: false,
		Index: 0,
		Hand:  []int{1, 2, 3},
	})
	m.store.SaveRoom(r)
	return r
}

func (m *Manager) AddBots(r *Room, n int) {
	for i := 0; i < n; i++ {
		r.Players = append(r.Players, Player{
			ID:    "bot-" + uuid.NewString(),
			Name:  "Bot",
			IsBot: true,
			Index: len(r.Players),
			Hand:  []int{3, 5, 7},
		})
	}
	m.store.SaveRoom(r)
}

func (m *Manager) Get(code string) (*Room, bool) {
	return m.store.GetRoom(code)
}

func (m *Manager) currentPlayer(r *Room) *Player {
	if len(r.Players) == 0 {
		return nil
	}
	return &r.Players[r.TurnIdx%len(r.Players)]
}

func (m *Manager) ApplyMove(r *Room, playerID string, x, y, card int) error {
	cp := m.currentPlayer(r)
	if cp == nil || cp.ID != playerID {
		return errors.New("not your turn or player invalid")
	}
	// ensure legal
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

	game.ApplyMove(&r.Board, x, y, playerID, card)
	// remove card from hand
	for i, v := range cp.Hand {
		if v == card {
			cp.Hand = append(cp.Hand[:i], cp.Hand[i+1:]...)
			break
		}
	}
	game.UpdateVState(&r.Board)
	if game.IsWinningAfter(r.Board, x, y, playerID, card) {
		r.WinnerID = &playerID
	}
	r.TurnIdx = (r.TurnIdx + 1) % len(r.Players)
	m.store.SaveRoom(r)
	return nil
}

func (m *Manager) BotMove(r *Room, botID string) (mv game.Move, err error) {
	cp := m.currentPlayer(r)
	if cp == nil || cp.ID != botID {
		return mv, errors.New("not bot's turn")
	}
	cands := game.GenerateLegalMoves(r.Board, cp.Hand, botID)
	if len(cands) == 0 {
		return mv, errors.New("no legal moves")
	}
	best := cands[0]
	bestScore := game.HeuristicScore(r.Board, best, r.Cfg)
	for _, c := range cands[1:] {
		if s := game.HeuristicScore(r.Board, c, r.Cfg); s > bestScore {
			best = c
			bestScore = s
		}
	}
	if err := m.ApplyMove(r, botID, best.X, best.Y, best.Card); err != nil {
		return mv, err
	}
	return best, nil
}

const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randCode(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type RankRow struct {
	PlayerID string `json:"playerId"`
	LineSum  int    `json:"tieBreakerLineSum"`
	TotalSum int    `json:"totalCellsSum"`
}

func (m *Manager) Rank(r *Room) []RankRow {
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
