// game.go
package game9x9

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

/*
  ========= DESIGN RINGKAS =========
  - Board: 9x9 of Cell {OwnerID, Value, Occupied}
  - Player: ID, Name, Color, Deck (queue), Hand (<=3), IsBot
  - Turn order: circular; first move forced to center (4,4 zero-based)
  - Adjacency rule: next move must be in Moore neighborhood of LAST MOVE
  - Overwrite rule: allowed if card > target.Value (self or opponent)
  - Win: 4 consecutive of same Owner (H/V/D) after a move
  - End: winner found OR no cards & no legal moves globally
  - Tie-break scoring (no 4-in-row): best contiguous segment sum (>=2) H/V/D
*/

const (
	BoardSize          = 9
	MaxHandSize        = 3
	CardMin, CardMax   = 1, 9
	CopiesPerValue     = 2
	NoOwner            = -1
	HugeWinScore       = 1_000_000_000
	BlockBigThreat     = 200_000
	MakeThreeBonus     = 5_000
	CaptureBonusFactor = 200
	CenterBonusFactor  = 50
)

type Pos struct{ R, C int }

type Cell struct {
	Owner int `json:"owner"` // player ID, -1 if empty
	Value int `json:"value"` // 0 if empty
}

type Player struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Deck   []int  `json:"-"`
	Hand   []int  `json:"hand"`
	IsBot  bool   `json:"is_bot"`
	Active bool   `json:"active"` // still can play (has cards OR may capture)
}

type Move struct {
	PlayerID int `json:"player_id"`
	R        int `json:"r"`    // 0..8
	C        int `json:"c"`    // 0..8
	Card     int `json:"card"` // 1..9
}

type Game struct {
	Board           [BoardSize][BoardSize]Cell `json:"board"`
	Players         []Player                   `json:"players"`
	TurnIdx         int                        `json:"turn_idx"` // index in Players slice
	LastMove        *Pos                       `json:"last_move,omitempty"`
	Started         bool                       `json:"started"`
	Finished        bool                       `json:"finished"`
	WinnerPlayerID  *int                       `json:"winner_player_id,omitempty"`
	MoveHistory     []Move                     `json:"move_history"`
	random          *rand.Rand
	initialFirstIdx int
}

// ===== Utilities

func NewDeck() []int {
	out := make([]int, 0, (CardMax-CardMin+1)*CopiesPerValue)
	for v := CardMin; v <= CardMax; v++ {
		for k := 0; k < CopiesPerValue; k++ {
			out = append(out, v)
		}
	}
	return out
}

func shuffle[T any](r *rand.Rand, a []T) {
	r.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
}

func NewGame(playerDefs []struct {
	Name  string
	Color string
	IsBot bool
}, seed int64) *Game {
	r := rand.New(rand.NewSource(seed))
	n := len(playerDefs)
	if n < 2 || n > 4 {
		panic("players must be 2..4")
	}
	players := make([]Player, n)
	idxs := r.Perm(n) // random order
	for i := 0; i < n; i++ {
		def := playerDefs[idxs[i]]
		deck := NewDeck()
		shuffle(r, deck)
		players[i] = Player{
			ID:     i,
			Name:   def.Name,
			Color:  def.Color,
			Deck:   deck,
			Hand:   []int{},
			IsBot:  def.IsBot,
			Active: true,
		}
	}
	g := &Game{
		Players:         players,
		TurnIdx:         0,
		Started:         false,
		Finished:        false,
		WinnerPlayerID:  nil,
		random:          r,
		initialFirstIdx: 0,
	}
	// initialize board
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			g.Board[r][c] = Cell{Owner: NoOwner, Value: 0}
		}
	}
	// initial draw to 3
	for i := range g.Players {
		g.drawToThree(i)
	}
	return g
}

func (g *Game) drawToThree(pi int) {
	p := &g.Players[pi]
	for len(p.Hand) < MaxHandSize && len(p.Deck) > 0 {
		// pop from deck head
		card := p.Deck[0]
		p.Deck = p.Deck[1:]
		p.Hand = append(p.Hand, card)
	}
}

func (g *Game) center() Pos { return Pos{R: BoardSize / 2, C: BoardSize / 2} }

func inBounds(r, c int) bool { return r >= 0 && r < BoardSize && c >= 0 && c < BoardSize }

var dirs = [][2]int{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1} /*X*/, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

func (g *Game) isAdjToLast(r, c int) bool {
	if g.LastMove == nil {
		// first move must be center
		cp := g.center()
		return r == cp.R && c == cp.C
	}
	l := *g.LastMove
	return max(abs(r-l.R), abs(c-l.C)) == 1 // Moore neighborhood
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ===== Rule checks & state transitions

// LegalMoves generates all legal (r,c,card) for a player index.
func (g *Game) LegalMoves(pi int) []Move {
	p := g.Players[pi]
	if len(p.Hand) == 0 {
		return nil
	}
	moves := []Move{}
	// First: candidate cells constrained by adjacency to last move
	var candidates []Pos
	if g.LastMove == nil {
		cp := g.center()
		candidates = []Pos{cp}
	} else {
		l := *g.LastMove
		for _, d := range dirs {
			r := l.R + d[0]
			c := l.C + d[1]
			if inBounds(r, c) {
				candidates = append(candidates, Pos{r, c})
			}
		}
	}

	for _, pos := range candidates {
		cell := g.Board[pos.R][pos.C]
		for _, card := range p.Hand {
			if cell.Owner == NoOwner {
				// empty: always ok
				moves = append(moves, Move{PlayerID: p.ID, R: pos.R, C: pos.C, Card: card})
			} else {
				// must be strictly greater
				if card > cell.Value {
					moves = append(moves, Move{PlayerID: p.ID, R: pos.R, C: pos.C, Card: card})
				}
			}
		}
	}
	return moves
}

func (g *Game) ApplyMove(m Move) error {
	if g.Finished {
		return errors.New("game already finished")
	}
	if m.PlayerID != g.Players[g.TurnIdx].ID {
		return errors.New("not this player's turn")
	}
	if !inBounds(m.R, m.C) {
		return errors.New("out of bounds")
	}
	if !g.isAdjToLast(m.R, m.C) {
		return errors.New("must place adjacent to last move (or center for first)")
	}
	// check hand contains card
	p := &g.Players[g.TurnIdx]
	cardIdx := -1
	for i, v := range p.Hand {
		if v == m.Card {
			cardIdx = i
			break
		}
	}
	if cardIdx == -1 {
		return errors.New("card not in hand")
	}
	// overwrite rule
	cell := g.Board[m.R][m.C]
	if cell.Owner != NoOwner && !(m.Card > cell.Value) {
		return errors.New("must be strictly greater to overwrite")
	}
	// place
	g.Board[m.R][m.C] = Cell{Owner: p.ID, Value: m.Card}
	g.LastMove = &Pos{R: m.R, C: m.C}
	// remove card from hand
	p.Hand = append(p.Hand[:cardIdx], p.Hand[cardIdx+1:]...)
	// draw
	g.drawToThree(g.TurnIdx)

	// history
	g.MoveHistory = append(g.MoveHistory, m)

	// check win
	if g.justMadeFour(m.R, m.C, p.ID) {
		g.Finished = true
		g.WinnerPlayerID = &p.ID
		return nil
	}

	// advance turn; also detect global end
	g.advanceTurnUntilPossibleOrEnd()
	return nil
}

func (g *Game) advanceTurnUntilPossibleOrEnd() {
	n := len(g.Players)
	loops := 0
	for {
		g.TurnIdx = (g.TurnIdx + 1) % n
		loops++
		if loops > n*2 {
			// safety
			break
		}
		if len(g.Players[g.TurnIdx].Hand) == 0 && len(g.Players[g.TurnIdx].Deck) == 0 {
			// likely no action; try next
			if g.anyoneCanMove() {
				continue
			}
			// End game by points
			g.finishByPoints()
			return
		}
		// if current has any legal moves -> stop
		if len(g.LegALMovesFast(g.TurnIdx)) > 0 {
			break
		}
		// if no one can move -> end
		if !g.anyoneCanMove() {
			g.finishByPoints()
			return
		}
		// else skip and continue
	}
}

func (g *Game) LegALMovesFast(pi int) []Move { // small alias, same as LegalMoves but faster path
	return g.LegalMoves(pi)
}

func (g *Game) anyoneCanMove() bool {
	for i := range g.Players {
		if len(g.LegALMovesFast(i)) > 0 {
			return true
		}
	}
	return false
}

func (g *Game) justMadeFour(r, c, owner int) bool {
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		count := 1
		// forward
		cr, cc := r+d[0], c+d[1]
		for inBounds(cr, cc) && g.Board[cr][cc].Owner == owner {
			count++
			cr += d[0]
			cc += d[1]
		}
		// backward
		cr, cc = r-d[0], c-d[1]
		for inBounds(cr, cc) && g.Board[cr][cc].Owner == owner {
			count++
			cr -= d[0]
			cc -= d[1]
		}
		if count >= 4 {
			return true
		}
	}
	return false
}

// finishByPoints: Rule 11 tiebreak
func (g *Game) finishByPoints() {
	bestSum := math.MinInt
	bestPID := -1
	for _, p := range g.Players {
		sum := g.bestSegmentSum(p.ID)
		if sum > bestSum {
			bestSum = sum
			bestPID = p.ID
		} else if sum == bestSum {
			// tie-break 1: total sum on board
			tp := g.totalSumFor(p.ID)
			tb := g.totalSumFor(bestPID)
			if tp > tb {
				bestPID = p.ID
			} else if tp == tb {
				// tie-break 2: lower turn index loses (arbitrary). Keep existing.
			}
		}
	}
	g.Finished = true
	if bestPID >= 0 {
		g.WinnerPlayerID = &bestPID
	}
}

func (g *Game) totalSumFor(pid int) int {
	sum := 0
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if g.Board[r][c].Owner == pid {
				sum += g.Board[r][c].Value
			}
		}
	}
	return sum
}

func (g *Game) bestSegmentSum(pid int) int {
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	best := 0
	anyPair := false
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			for _, d := range dirs {
				sum := 0
				len := 0
				cr, cc := r, c
				for inBounds(cr, cc) && g.Board[cr][cc].Owner == pid {
					sum += g.Board[cr][cc].Value
					len++
					cr += d[0]
					cc += d[1]
				}
				if len >= 2 {
					anyPair = true
					if sum > best {
						best = sum
					}
				}
			}
		}
	}
	if anyPair {
		return best
	}
	// tidak ada segmen >=2, ambil nilai tunggal terbesar
	solo := 0
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if g.Board[r][c].Owner == pid && g.Board[r][c].Value > solo {
				solo = g.Board[r][c].Value
			}
		}
	}
	return solo
}

// ===== Bot AI

func (g *Game) BotChooseMove(pi int) (Move, bool) {
	candidates := g.LegalMoves(pi)
	if len(candidates) == 0 {
		return Move{}, false
	}
	best := candidates[0]
	bestScore := math.MinInt
	for _, mv := range candidates {
		score := g.evaluateMove(pi, mv)
		if score > bestScore {
			bestScore = score
			best = mv
		}
	}
	return best, true
}

func (g *Game) evaluateMove(pi int, mv Move) int {
	// simulate on a light copy
	copyG := g.shallowCopy()
	_ = copyG.ApplyMoveNoTurnAdvance(mv) // ignore errors; we know it's legal
	// 1) Win now?
	if copyG.justMadeFour(mv.R, mv.C, pi) {
		return HugeWinScore + mv.Card
	}

	// 2) Block opponent big threat (rough heuristic):
	blockScore := 0
	origThreat := g.maxOpponentThreat(pi)
	newThreat := copyG.maxOpponentThreat(pi)
	if newThreat < origThreat {
		blockScore += BlockBigThreat * (origThreat - newThreat)
	}

	// 3) Build own segment score through (r,c)
	buildLen, buildSum := copyG.longestThrough(mv.R, mv.C, pi)
	buildScore := buildLen*1000 + buildSum*50
	if buildLen == 3 {
		buildScore += MakeThreeBonus
	}

	// 4) Capture bonus
	prev := g.Board[mv.R][mv.C]
	capBonus := 0
	if prev.Owner != NoOwner {
		capBonus += (mv.Card - prev.Value) * CaptureBonusFactor
	}

	// 5) Centrality
	center := g.center()
	dist := max(abs(mv.R-center.R), abs(mv.C-center.C))
	centerBonus := (BoardSize/2 - dist) * CenterBonusFactor

	return blockScore + buildScore + capBonus + centerBonus + mv.Card
}

func (g *Game) ApplyMoveNoTurnAdvance(m Move) error {
	// Like ApplyMove but no turn advance and no draw (for evaluation only)
	if !inBounds(m.R, m.C) {
		return errors.New("OOB")
	}
	if !g.isAdjToLast(m.R, m.C) {
		return errors.New("adj")
	}
	cell := g.Board[m.R][m.C]
	if cell.Owner != NoOwner && !(m.Card > cell.Value) {
		return errors.New("overwrite rule")
	}
	g.Board[m.R][m.C] = Cell{Owner: m.PlayerID, Value: m.Card}
	g.LastMove = &Pos{R: m.R, C: m.C}
	return nil
}

func (g *Game) shallowCopy() *Game {
	cp := *g
	// deep copy board
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			cp.Board[r][c] = g.Board[r][c]
		}
	}
	if g.LastMove != nil {
		lm := *g.LastMove
		cp.LastMove = &lm
	}
	return &cp
}

func (g *Game) maxOpponentThreat(myID int) int {
	best := 0
	for _, p := range g.Players {
		if p.ID == myID {
			continue
		}
		if v := g.maxLenAnywhere(p.ID); v > best {
			best = v
		}
	}
	return best
}

func (g *Game) maxLenAnywhere(pid int) int {
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	best := 0
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			for _, d := range dirs {
				len := 0
				cr, cc := r, c
				for inBounds(cr, cc) && g.Board[cr][cc].Owner == pid {
					len++
					cr += d[0]
					cc += d[1]
				}
				if len > best {
					best = len
				}
			}
		}
	}
	return best
}

func (g *Game) longestThrough(r, c, pid int) (length int, sum int) {
	// Compute best line length & sum passing through (r,c) for pid
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	bestLen, bestSum := 1, g.Board[r][c].Value
	for _, d := range dirs {
		// go one side
		len1, sum1 := 0, 0
		cr, cc := r-d[0], c-d[1]
		for inBounds(cr, cc) && g.Board[cr][cc].Owner == pid {
			len1++
			sum1 += g.Board[cr][cc].Value
			cr -= d[0]
			cc -= d[1]
		}
		// other side
		len2, sum2 := 0, 0
		cr, cc = r+d[0], c+d[1]
		for inBounds(cr, cc) && g.Board[cr][cc].Owner == pid {
			len2++
			sum2 += g.Board[cr][cc].Value
			cr += d[0]
			cc += d[1]
		}
		totalLen := 1 + len1 + len2
		totalSum := g.Board[r][c].Value + sum1 + sum2
		if totalLen > bestLen || (totalLen == bestLen && totalSum > bestSum) {
			bestLen, bestSum = totalLen, totalSum
		}
	}
	return bestLen, bestSum
}

// ===== Simple text driver (optional) =====

type PublicState struct {
	Board          [BoardSize][BoardSize]Cell `json:"board"`
	Players        []Player                   `json:"players"`
	TurnIdx        int                        `json:"turn_idx"`
	LastMove       *Pos                       `json:"last_move,omitempty"`
	Started        bool                       `json:"started"`
	Finished       bool                       `json:"finished"`
	WinnerPlayerID *int                       `json:"winner_player_id,omitempty"`
	MoveHistory    []Move                     `json:"move_history"`
}

func (g *Game) Export() PublicState {
	return PublicState{
		Board:          g.Board,
		Players:        g.Players,
		TurnIdx:        g.TurnIdx,
		LastMove:       g.LastMove,
		Started:        g.Started,
		Finished:       g.Finished,
		WinnerPlayerID: g.WinnerPlayerID,
		MoveHistory:    g.MoveHistory,
	}
}

// Demo CLI — jalankan dengan go run . pada package main kalian sendiri.
// Di sini aku sisipkan fungsi main untuk simulasi cepat.
func demo() {
	pdefs := []struct {
		Name  string
		Color string
		IsBot bool
	}{
		{"Alice", "green", false}, // bisa kalian ganti jadi bot juga
		{"BotX", "red", true},
	}
	g := NewGame(pdefs, time.Now().UnixNano())

	// Start flag
	g.Started = true

	// Loop sampai selesai, di demo ini semua pemain otomatis jadi bot.
	for !g.Finished {
		mv, ok := g.BotChooseMove(g.TurnIdx)
		if !ok {
			// skip
			g.advanceTurnUntilPossibleOrEnd()
			continue
		}
		_ = g.ApplyMove(mv)

		// Cetak ringkas
		js, _ := json.Marshal(struct {
			Last Move `json:"last"`
		}{Last: mv})
		fmt.Println(string(js))
	}

	out, _ := json.MarshalIndent(g.Export(), "", "  ")
	fmt.Println(string(out))
}

// Uncomment untuk menjalankan demo sebagai program standalone.
// func main() { demo() }
