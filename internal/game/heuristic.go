package game

import (
	"javanese-chess/internal/config"
	"log"
)

// EvaluateMove calculates the heuristic score for a move
// Based on the heuristic value table provided
func EvaluateMove(b *Board, x, y int, card int, playerID string, cfg *config.Config) int {
	weights := cfg.DefaultWeights
	score := 0

	// Base value: Legal move
	score += weights.LegalMove // 30

	// 1. f_win: Winning move (4-in-a-row)
	if f_win(b, x, y, playerID, card) {
		winScore := weights.WWin // 10000
		score += winScore
		log.Printf("Move (%d,%d) card=%d | f_win=%d", x, y, card, winScore)
		return score // If winning, return immediately
	}

	// 2. f_threat: Detect if opponent has 3-in-a-row and this blocks it
	isThreat := f_threat(b, x, y, playerID)
	threatScore := 0
	if isThreat {
		threatScore = weights.WThreat // 200
		score += threatScore
	}

	// 3. f_replace: Replace opponent's card
	replaceScore := f_replace(b, x, y, playerID, isThreat, &weights)
	score += replaceScore

	// 4. f_blocks: Block opponent's path
	blocksScore := f_blocks(b, x, y, playerID, isThreat, &weights)
	score += blocksScore

	// 5. f_formation: Build our own alignments
	formationScore := f_formation(b, x, y, playerID, card, &weights)
	score += formationScore

	// 6. f_value: Card value management
	valueScore := f_value(b, x, y, card, playerID, isThreat, &weights)
	score += valueScore

	// 7. Play smallest card bonus
	// This is handled inside f_value

	// 8. Place card close to our own cards
	proximityScore := f_proximity(b, x, y, playerID, &weights)
	score += proximityScore

	log.Printf("Move (%d,%d) card=%d | threat=%d replace=%d blocks=%d formation=%d value=%d proximity=%d | TOTAL=%d",
		x, y, card, threatScore, replaceScore, blocksScore, formationScore, valueScore, proximityScore, score)

	return score
}

// f_win: Returns true if placing card at (x,y) creates 4-in-a-row
func f_win(b *Board, x, y int, playerID string, card int) bool {
	// Temporarily place the card
	originalOwner := b.Cells[y][x].OwnerID
	originalValue := b.Cells[y][x].Value

	b.Cells[y][x].OwnerID = playerID
	b.Cells[y][x].Value = card

	// Check if this creates 4-in-a-row
	hasWin := check4InARow(b, x, y, playerID)

	// Restore original state
	b.Cells[y][x].OwnerID = originalOwner
	b.Cells[y][x].Value = originalValue

	return hasWin
}

// check4InARow checks if there are 4 cards in a row for playerID at position (x,y)
func check4InARow(b *Board, x, y int, playerID string) bool {
	directions := [][2]int{
		{1, 0},  // Horizontal
		{0, 1},  // Vertical
		{1, 1},  // Diagonal down-right
		{1, -1}, // Diagonal up-right
	}

	for _, dir := range directions {
		count := 1 // Count the current cell

		// Check forward direction
		nx, ny := x+dir[0], y+dir[1]
		for in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID == playerID {
			count++
			nx += dir[0]
			ny += dir[1]
		}

		// Check backward direction
		nx, ny = x-dir[0], y-dir[1]
		for in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID == playerID {
			count++
			nx -= dir[0]
			ny -= dir[1]
		}

		if count >= 4 {
			return true
		}
	}

	return false
}

// f_threat: Returns true if opponent has 3-in-a-row and (x,y) blocks it
func f_threat(b *Board, x, y int, playerID string) bool {
	// Get all opponent IDs
	opponents := getOpponentIDs(b, playerID)

	// Check if any opponent has 3-in-a-row that would be blocked by this move
	for _, opponentID := range opponents {
		if blocks3InARow(b, x, y, opponentID) {
			return true
		}
	}

	return false
}

// blocks3InARow checks if placing at (x,y) blocks opponent's 3-in-a-row
func blocks3InARow(b *Board, x, y int, opponentID string) bool {
	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	for _, dir := range directions {
		// Check if this position is part of a potential 4-in-a-row for opponent
		// We need to check if opponent has 3 cards in a line and (x,y) is the 4th position
		for offset := -3; offset <= 0; offset++ {
			opponentCount := 0
			emptyCount := 0
			valid := true

			for i := 0; i < 4; i++ {
				px := x + dir[0]*(offset+i)
				py := y + dir[1]*(offset+i)

				if !in(px, py, b.Size) {
					valid = false
					break
				}

				if px == x && py == y {
					emptyCount++
					continue
				}

				cell := b.Cells[py][px]
				if cell.OwnerID == opponentID {
					opponentCount++
				} else if cell.OwnerID == "" {
					emptyCount++
				}
			}

			// If opponent has 3 cards and (x,y) is the only empty spot, it's a threat
			if valid && opponentCount == 3 && emptyCount == 1 {
				return true
			}
		}
	}

	return false
}

// f_replace: Score for replacing opponent's card
func f_replace(b *Board, x, y int, playerID string, isThreat bool, weights *config.HeuristicWeights) int {
	cell := b.Cells[y][x]

	// If empty or own card, no replacement score
	if cell.OwnerID == "" || cell.OwnerID == playerID {
		return 0
	}

	// Base replacement value depends on threat context
	replaceValue := 0
	if isThreat {
		replaceValue = weights.ReplaceWhenThreat // 200
	} else {
		replaceValue = weights.ReplacePotential // 125
	}

	// Add position bonus
	positionBonus := getPositionBonus(b, x, y, cell.OwnerID, weights)
	
	return replaceValue + positionBonus
}

// getPositionBonus calculates bonus based on position in opponent's line
func getPositionBonus(b *Board, x, y int, opponentID string, weights *config.HeuristicWeights) int {
	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	maxBonus := 0

	for _, dir := range directions {
		// Count cards in both directions
		backCount := countConsecutive(b, x, y, -dir[0], -dir[1], opponentID)
		forwardCount := countConsecutive(b, x, y, dir[0], dir[1], opponentID)

		lineLength := backCount + forwardCount + 1

		if lineLength >= 3 {
			// Determine if center or side
			if backCount >= 1 && forwardCount >= 1 {
				// Center position (cards on both sides)
				bonus := weights.ReplacePosCenter // 75
				if bonus > maxBonus {
					maxBonus = bonus
				}
			} else {
				// Side position (cards only on one side)
				bonus := weights.ReplacePosSide // 50
				if bonus > maxBonus {
					maxBonus = bonus
				}
			}
		}
	}

	return maxBonus
}

// countConsecutive counts consecutive cards of owner in a direction
func countConsecutive(b *Board, x, y int, dx, dy int, ownerID string) int {
	count := 0
	nx, ny := x+dx, y+dy

	for in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID == ownerID {
		count++
		nx += dx
		ny += dy
	}

	return count
}

// f_blocks: Score for blocking opponent's path
func f_blocks(b *Board, x, y int, playerID string, isThreat bool, weights *config.HeuristicWeights) int {
	maxBlockScore := 0

	opponents := getOpponentIDs(b, playerID)

	for _, opponentID := range opponents {
		// Check if this blocks a 3-in-a-row (immediate threat)
		if blocks3InARow(b, x, y, opponentID) {
			blockScore := weights.BlockWhenThreat // 100
			if blockScore > maxBlockScore {
				maxBlockScore = blockScore
			}
		} else if blocks2InARow(b, x, y, opponentID) {
			// Check if this blocks a 2-in-a-row (potential threat)
			blockScore := weights.BlockPotential // 70
			if blockScore > maxBlockScore {
				maxBlockScore = blockScore
			}
		}
	}

	return maxBlockScore
}

// blocks2InARow checks if placing at (x,y) blocks opponent's 2-in-a-row extension
func blocks2InARow(b *Board, x, y int, opponentID string) bool {
	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	for _, dir := range directions {
		// Check if opponent has 2 cards in a line and (x,y) could extend it
		backCount := countConsecutive(b, x, y, -dir[0], -dir[1], opponentID)
		forwardCount := countConsecutive(b, x, y, dir[0], dir[1], opponentID)

		totalCount := backCount + forwardCount

		if totalCount >= 2 {
			return true
		}
	}

	return false
}

// f_formation: Score for building our own alignments
func f_formation(b *Board, x, y int, playerID string, card int, weights *config.HeuristicWeights) int {
	// Temporarily place the card
	originalOwner := b.Cells[y][x].OwnerID
	originalValue := b.Cells[y][x].Value

	b.Cells[y][x].OwnerID = playerID
	b.Cells[y][x].Value = card

	maxAlignment := getMaxAlignment(b, x, y, playerID)

	// Restore original state
	b.Cells[y][x].OwnerID = originalOwner
	b.Cells[y][x].Value = originalValue

	if maxAlignment >= 3 {
		return weights.BuildAlignment3 // 100
	} else if maxAlignment >= 2 {
		return weights.BuildAlignment2 // 50
	}

	return 0
}

// getMaxAlignment returns the maximum consecutive cards in any direction
func getMaxAlignment(b *Board, x, y int, playerID string) int {
	directions := [][2]int{
		{1, 0}, {0, 1}, {1, 1}, {1, -1},
	}

	maxCount := 1

	for _, dir := range directions {
		count := 1
		count += countConsecutive(b, x, y, dir[0], dir[1], playerID)
		count += countConsecutive(b, x, y, -dir[0], -dir[1], playerID)

		if count > maxCount {
			maxCount = count
		}
	}

	return maxCount
}

// f_value: Card value management based on context
func f_value(b *Board, x, y int, card int, playerID string, isThreat bool, weights *config.HeuristicWeights) int {
	cell := b.Cells[y][x]
	isReplacingOpponent := cell.OwnerID != "" && cell.OwnerID != playerID

	// Determine card value based on context
	cardValue := 0
	if isThreat && isReplacingOpponent {
		// Blocking threat: prefer high cards (Card 9 = 100, Card 1 = 20)
		cardValue = weights.ReplaceValuesThreat[card]
	} else {
		// Defensive play: prefer low cards (Card 1 = 100, Card 9 = 20)
		cardValue = weights.ReplaceValuesPotential[card]
	}

	return cardValue
}

// f_proximity: Bonus for placing card close to our own cards
func f_proximity(b *Board, x, y int, playerID string, weights *config.HeuristicWeights) int {
	// Check if there are any adjacent cards owned by the player
	directions := [][2]int{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
	}

	for _, dir := range directions {
		nx, ny := x+dir[0], y+dir[1]
		if in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID == playerID {
			return weights.KeepNearCard // 60
		}
	}

	return 0
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

// Helper function: in checks if coordinates are within board bounds
func in(x, y, n int) bool {
	return x >= 0 && y >= 0 && x < n && y < n
}
