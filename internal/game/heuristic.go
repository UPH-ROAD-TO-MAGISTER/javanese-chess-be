package game

import (
	"javanese-chess/internal/config"
	"log"
)

// HeuristicScore evaluates a move based on the heuristic function from Section 2.4
// H(s,a) = W₁·f_win + W₂·f_threat_block + W₃·f_replace_value + W₄·f_block_path + W₅·f_build_alignment + W₆·f_card_cost
func HeuristicScore(
	b Board,
	move Move,
	hand []int,
	weights config.HeuristicWeights,
) int {
	score := 0

	// Create hypothetical board after this move
	hypoBoard := cloneBoard(b)
	ApplyMove(&hypoBoard, move.X, move.Y, move.PlayerID, move.Card)

	// W₁: f_win - Immediate winning move (4-in-a-row)
	if IsWinningAfter(hypoBoard, move.X, move.Y, move.PlayerID, move.Card) {
		score += weights.WWin
	}

	// W₂: f_threat_block - Blocking opponent's immediate threat
	threatScore := evaluateThreatBlock(b, move)
	score += weights.WThreat * threatScore

	// W₃: f_replace_value - Overwriting opponent cards
	replaceScore := evaluateReplaceValue(b, move)
	score += weights.WReplaceValue * replaceScore / 100

	// W₄: f_block_path - Blocking enemy paths
	blockPathScore := evaluateBlockPath(b, move)
	score += weights.WBlockPath * blockPathScore

	// W₅: f_build_alignment - Building own alignments
	buildScore := evaluateBuildAlignment(hypoBoard, move)
	score += weights.WBuildAlignment * buildScore / 10

	// W₆: f_card_cost - Card value management
	cardCostScore := evaluateCardCost(b, move, hand)
	score += weights.WCardCost * cardCostScore

	return score
}

// cloneBoard creates a deep copy of the board
func cloneBoard(b Board) Board {
	newBoard := NewBoard(b.Size)
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			newBoard.Cells[y][x] = b.Cells[y][x]
		}
	}
	return newBoard
}

// evaluateThreatBlock implements f_threat_block from Section 2.4.2
// Returns 1 if this move blocks an opponent's 3-in-a-row threat, 0 otherwise
func evaluateThreatBlock(b Board, move Move) int {
	// Check if this move blocks any opponent's threat
	if blocksImmediateThreat(b, move.X, move.Y, move.PlayerID) {
		return 1
	}
	return 0
}

// evaluateReplaceValue implements f_replace_value from Section 2.4.3
// Returns score based on overwriting opponent cards
func evaluateReplaceValue(b Board, move Move) int {
	oldCell := b.Cells[move.Y][move.X]

	// Not replacing anything
	if oldCell.OwnerID == "" || oldCell.OwnerID == move.PlayerID {
		return 0
	}

	// Check if this is responding to a threat
	if blocksImmediateThreat(b, move.X, move.Y, move.PlayerID) {
		// Replacing as threat response: 200 + midpoint bonus
		midpointBonus := evaluateMidpointBonus(b, move, oldCell.OwnerID)
		return 200 + midpointBonus
	}

	// General replacement (not urgent): 125
	return 125
}

// evaluateMidpointBonus calculates bonus for replacing middle of opponent's line
// Returns 75 for midpoint, 50 for edge, 0 otherwise
func evaluateMidpointBonus(b Board, move Move, opponentID string) int {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}

	for _, dir := range dirs {
		// Check backward
		backCount := countInDirection(b, move.X, move.Y, -dir[0], -dir[1], opponentID)
		// Check forward
		forwardCount := countInDirection(b, move.X, move.Y, dir[0], dir[1], opponentID)

		lineLength := backCount + forwardCount + 1

		if lineLength >= 3 {
			// Determine position in line
			if backCount >= 1 && forwardCount >= 1 {
				return 75 // Middle of line
			}
			return 50 // Edge of line
		}
	}
	return 0
}

// evaluateBlockPath implements f_block_path from Section 2.4.4
// Returns 1 if move cuts opponent's path, 0 otherwise
func evaluateBlockPath(b Board, move Move) int {
	// Check if current cell is empty
	if b.Cells[move.Y][move.X].OwnerID != "" {
		return 0
	}

	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}

	// Check all directions for opponent cards on both sides
	for _, dir := range dirs {
		x1, y1 := move.X+dir[0], move.Y+dir[1]
		x2, y2 := move.X-dir[0], move.Y-dir[1]

		if in(x1, y1, b.Size) && in(x2, y2, b.Size) {
			owner1 := b.Cells[y1][x1].OwnerID
			owner2 := b.Cells[y2][x2].OwnerID

			// Check if both sides have opponent cards (same owner, different from player)
			if owner1 != "" && owner1 == owner2 && owner1 != move.PlayerID {
				return 1
			}
		}
	}
	return 0
}

// evaluateBuildAlignment implements f_build_alignment from Section 2.4.5
// Returns 100 for 3-in-a-row, 50 for 2-in-a-row, 0 otherwise
func evaluateBuildAlignment(b Board, move Move) int {
	maxAlignment := chainLenAfter(b, move.X, move.Y, move.PlayerID)

	// Scoring based on alignment length
	if maxAlignment >= 3 {
		return 100
	} else if maxAlignment == 2 {
		return 50
	}
	return 0
}

// evaluateCardCost implements f_card_cost from Section 2.4.6
// Returns score based on card value management strategy
func evaluateCardCost(b Board, move Move, hand []int) int {
	cardValue := move.Card

	// Determine if this is a threat response
	isThreatResponse := blocksImmediateThreat(b, move.X, move.Y, move.PlayerID)

	// Base score from table (Section 2.4.6)
	var baseScore int
	if isThreatResponse {
		// Threat Response: higher values preferred
		baseScore = cardValue*10 + 20
	} else {
		// Potential threat: lower values preferred
		baseScore = (10 - cardValue) * 10
	}

	// Smallest card bonus: +60 if this is the lowest card in hand
	if isSmallestInHand(cardValue, hand) {
		baseScore += 60
	}

	return baseScore
}

// Helper: Check if card is smallest in current hand
func isSmallestInHand(cardValue int, hand []int) bool {
	for _, c := range hand {
		if c < cardValue {
			return false
		}
	}
	return true
}

// Helper: Count cards in a direction from position
func countInDirection(b Board, x, y, dx, dy int, ownerID string) int {
	count := 0
	nx, ny := x+dx, y+dy

	for in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID == ownerID {
		count++
		nx += dx
		ny += dy
	}
	return count
}

// getOpponentIDs returns all opponent player IDs on the board
func getOpponentIDs(b *Board, playerID string) []string {
	seen := make(map[string]bool)
	var opponents []string

	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			ownerID := b.Cells[y][x].OwnerID
			if ownerID != "" && ownerID != playerID && !seen[ownerID] {
				seen[ownerID] = true
				opponents = append(opponents, ownerID)
			}
		}
	}

	return opponents
}

// countAdjacentOpponentCards counts opponent cards in 8 directions
func countAdjacentOpponentCards(b *Board, x, y int, playerID string) int {
	count := 0

	for q := -1; q <= 1; q++ {
		for p := -1; p <= 1; p++ {
			if p == 0 && q == 0 {
				continue
			}
			nx, ny := x+p, y+q
			if nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
				cell := b.Cells[ny][nx]
				if cell.OwnerID != "" && cell.OwnerID != playerID {
					count++
				}
			}
		}
	}

	return count
}

// countLineOpponentCards counts consecutive opponent cards in a direction
func countLineOpponentCards(b *Board, x, y int, dir [2]int, playerID string) int {
	count := 0

	// Forward direction
	nx, ny := x+dir[0], y+dir[1]
	for nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
		cell := b.Cells[ny][nx]
		if cell.OwnerID != "" && cell.OwnerID != playerID {
			count++
			nx += dir[0]
			ny += dir[1]
		} else {
			break
		}
	}

	// Backward direction
	nx, ny = x-dir[0], y-dir[1]
	for nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
		cell := b.Cells[ny][nx]
		if cell.OwnerID != "" && cell.OwnerID != playerID {
			count++
			nx -= dir[0]
			ny -= dir[1]
		} else {
			break
		}
	}

	return count
}

// countLineOwnCards counts consecutive own cards in a direction
func countLineOwnCards(b *Board, x, y int, dir [2]int, playerID string) int {
	count := 1 // The position itself

	// Forward direction
	nx, ny := x+dir[0], y+dir[1]
	for nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
		if b.Cells[ny][nx].OwnerID == playerID {
			count++
			nx += dir[0]
			ny += dir[1]
		} else {
			break
		}
	}

	// Backward direction
	nx, ny = x-dir[0], y-dir[1]
	for nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
		if b.Cells[ny][nx].OwnerID == playerID {
			count++
			nx -= dir[0]
			ny -= dir[1]
		} else {
			break
		}
	}

	return count
}

// hasThreeInRowThreat checks if opponent has 3-in-a-row at this position
func hasThreeInRowThreat(b *Board, x, y int, opponentID string) bool {
	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	for _, dir := range directions {
		count := 1 // The position itself

		// Forward direction
		nx, ny := x+dir[0], y+dir[1]
		for i := 0; i < 3; i++ {
			if nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
				if b.Cells[ny][nx].OwnerID == opponentID {
					count++
					nx += dir[0]
					ny += dir[1]
				} else {
					break
				}
			} else {
				break
			}
		}

		// Backward direction
		nx, ny = x-dir[0], y-dir[1]
		for i := 0; i < 3; i++ {
			if nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
				if b.Cells[ny][nx].OwnerID == opponentID {
					count++
					nx -= dir[0]
					ny -= dir[1]
				} else {
					break
				}
			} else {
				break
			}
		}

		if count >= 3 {
			return true
		}
	}

	return false
}

// f_win detects if move creates 4-in-a-row (winning move)
func f_win(b *Board, x, y int, playerID string) int {
	// Temporarily place the card
	originalOwner := b.Cells[y][x].OwnerID
	b.Cells[y][x].OwnerID = playerID

	// Check if this creates a win
	hasWin := CheckWin(b, playerID)

	// Restore original state
	b.Cells[y][x].OwnerID = originalOwner

	if hasWin {
		return 1
	}
	return 0
}

// f_threat detects opponent threats at position
func f_threat(b *Board, x, y int, playerID string) int {
	opponents := getOpponentIDs(b, playerID)

	for _, opponentID := range opponents {
		if hasThreeInRowThreat(b, x, y, opponentID) {
			return 1
		}
	}

	return 0
}

// f_replace evaluates replacement value during threats
func f_replace(b *Board, x, y int, playerID string, isThreat bool, cfg *config.Config) int {
	cell := b.Cells[y][x]

	// Only score if overwriting opponent during threat
	if !isThreat || cell.OwnerID == "" || cell.OwnerID == playerID {
		return 0
	}

	score := cfg.DefaultWeights.ReplaceWhenThreat // Base: 200

	// Add positional bonus based on adjacent opponent cards
	adjacentOpponentCount := countAdjacentOpponentCards(b, x, y, playerID)

	if adjacentOpponentCount >= 3 {
		score += cfg.DefaultWeights.ReplacePosMiddle // +75 (middle position)
	} else if adjacentOpponentCount >= 1 {
		score += cfg.DefaultWeights.ReplacePosSide // +50 (side position)
	}

	return score
}

// f_blocks calculates blocking value for adjacent opponent cards
func f_blocks(b *Board, x, y int, playerID string, cfg *config.Config) int {
	score := 0

	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	for _, dir := range directions {
		count := countLineOpponentCards(b, x, y, dir, playerID)

		if count >= 3 {
			score += cfg.DefaultWeights.BlockWhenThreat // +100
		} else if count >= 2 {
			score += cfg.DefaultWeights.BlockPotential // +70
		}
	}

	return score
}

// f_formation rewards building 2 or 3-in-a-row alignments
func f_formation(b *Board, x, y int, playerID string, cfg *config.Config) int {
	maxAlignment := 0

	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	for _, dir := range directions {
		count := countLineOwnCards(b, x, y, dir, playerID)
		if count > maxAlignment {
			maxAlignment = count
		}
	}

	if maxAlignment >= 3 {
		return cfg.DefaultWeights.BuildAlignment3 // +100
	} else if maxAlignment >= 2 {
		return cfg.DefaultWeights.BuildAlignment2 // +50
	}

	return 0
}

// f_value assesses card value (dual role: general play vs threat response)
func f_value(card int, isOverwritingDuringThreat bool, cfg *config.Config) int {
	if isOverwritingDuringThreat {
		// Use high cards when responding to threats
		return cfg.DefaultWeights.ReplaceValuesThreat[card]
	}
	// Use low cards in general play (save high cards)
	return cfg.DefaultWeights.ReplaceValuesPotential[card]
}

func EvaluateMove(b *Board, x, y int, card int, playerID string, cfg *config.Config) int {
	score := 0

	// 1. f_win
	winScore := 0
	if f_win(b, x, y, playerID) == 1 {
		winScore = cfg.DefaultWeights.WWin
		score += winScore
	}

	// 2. f_threat
	threatScore := 0
	isThreat := f_threat(b, x, y, playerID) == 1
	if isThreat {
		threatScore = cfg.DefaultWeights.WThreat
		score += threatScore
	}

	// 3. f_replace
	replaceScore := f_replace(b, x, y, playerID, isThreat, cfg)
	score += replaceScore

	// 4. f_blocks
	blocksScore := f_blocks(b, x, y, playerID, cfg)
	score += blocksScore

	// 5. f_formation
	formationScore := f_formation(b, x, y, playerID, cfg)
	score += formationScore

	// 6. f_value
	cell := b.Cells[y][x]
	isOverwritingDuringThreat := isThreat && cell.OwnerID != "" && cell.OwnerID != playerID
	valueScore := f_value(card, isOverwritingDuringThreat, cfg)
	score += valueScore

	// Debug logging
	log.Printf("Move (%d,%d) card=%d | win=%d threat=%d replace=%d blocks=%d formation=%d value=%d | TOTAL=%d",
		x, y, card, winScore, threatScore, replaceScore, blocksScore, formationScore, valueScore, score)

	return score
}

// CheckWin checks if a player has 4 cards in a row
func CheckWin(b *Board, playerID string) bool {
	directions := [][2]int{
		{1, 0},  // Horizontal
		{0, 1},  // Vertical
		{1, 1},  // Diagonal down-right
		{1, -1}, // Diagonal up-right
	}

	// Check every position on the board
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			if b.Cells[y][x].OwnerID != playerID {
				continue
			}

			// Check each direction from this position
			for _, dir := range directions {
				count := 1 // Count the current cell

				// Check forward direction
				nx, ny := x+dir[0], y+dir[1]
				for count < 4 && nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
					if b.Cells[ny][nx].OwnerID == playerID {
						count++
						nx += dir[0]
						ny += dir[1]
					} else {
						break
					}
				}

				if count >= 4 {
					return true
				}
			}
		}
	}

	return false
}
