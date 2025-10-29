package game

import (
	"javanese-chess/internal/config"
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
