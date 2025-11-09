package game

import "log"

type ThreatType int

const (
	ThreatNone ThreatType = iota
	ThreatImmediate
)

type Threat struct {
	Type ThreatType
	X, Y int
	Dir  [2]int
}

func TieBreakerLineSum(b Board, playerID string) int {
	maxSum := 0
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			if b.Cells[y][x].OwnerID != playerID {
				continue
			}
			for _, d := range dirs {
				sum := b.Cells[y][x].Value
				px, py := x+d[0], y+d[1]
				for in(px, py, b.Size) && b.Cells[py][px].OwnerID == playerID {
					sum += b.Cells[py][px].Value
					px += d[0]
					py += d[1]
				}
				if sum > maxSum {
					maxSum = sum
				}
			}
		}
	}
	return maxSum
}

func TotalOwnedSum(b Board, playerID string) int {
	sum := 0
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			if b.Cells[y][x].OwnerID == playerID {
				sum += b.Cells[y][x].Value
			}
		}
	}
	return sum
}

// GenerateLegalMoves generates all legal moves for a player
func GenerateLegalMoves(b *Board, hand []int, playerID string) []Move {
	var moves []Move

	// Check if this is the first move of the game (board is empty)
	boardEmpty := true
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			if b.Cells[y][x].Value != 0 {
				boardEmpty = false
				break
			}
		}
		if !boardEmpty {
			break
		}
	}

	// RULE: First move must be at center position [4,4] (0-indexed)
	if boardEmpty {
		centerX, centerY := b.Size/2, b.Size/2 // For 9x9 board: [4,4]
		for _, card := range hand {
			moves = append(moves, Move{X: centerX, Y: centerY, Card: card, PlayerID: playerID})
		}
		// Debug log
		if len(moves) > 0 {
			log.Printf("DEBUG: First move detected. Board empty. Center: (%d,%d). Generated %d moves", centerX, centerY, len(moves))
		}
		return moves
	}

	// Regular move generation (after first move)
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			cell := b.Cells[y][x]

			// Skip cells that are not adjacent to any placed card
			// Only allow placement on:
			// 1. Empty cells adjacent to filled cells (CellBlocked = 1)
			// 2. Filled cells that can be replaced (CellReplaceable = 2)
			if cell.VState == CellAccessible && cell.Value == 0 {
				// Empty cell with no neighbors - NOT ALLOWED
				continue
			}

			// Skip permanent card 9 (cannot overwrite)
			if cell.Value == 9 {
				continue
			}

			for _, card := range hand {
				// If cell is empty (CellBlocked), any card can be placed
				if cell.Value == 0 {
					moves = append(moves, Move{X: x, Y: y, Card: card, PlayerID: playerID})
					continue
				}

				// If cell is filled (CellReplaceable):
				// - Card must be higher than current value
				// - Cannot overwrite own card
				if cell.Value >= card {
					continue
				}
				if cell.OwnerID == playerID {
					continue
				}

				moves = append(moves, Move{X: x, Y: y, Card: card, PlayerID: playerID})
			}
		}
	}

	return moves
}
