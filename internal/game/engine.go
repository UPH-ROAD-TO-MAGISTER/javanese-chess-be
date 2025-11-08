package game

import (
	"errors"
	"javanese-chess/internal/config"
)

func ApplyMove(b *Board, x, y int, owner string, card int) {
	cell := &b.Cells[y][x]
	cell.OwnerID = owner
	cell.Value = card

	// Update virtual states after placement
	UpdateLocalVState(b, x, y)
}

// UpdateVState updates virtual states for all cells on the board
func UpdateVState(b *Board) {
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			cell := &b.Cells[y][x]

			// Rule 3: Card 9 is permanent
			if cell.Value == 9 {
				cell.VState = CellAccessible // v(x,y) = 0
				continue
			}

			// Rule 2: Filled cells are replaceable
			if cell.Value != 0 {
				cell.VState = CellReplaceable // v(x,y) = 2
				continue
			}

			// Rule 1: Empty cells with filled neighbors are blocked
			if hasFilledNeighbor(b, x, y) {
				cell.VState = CellBlocked // v(x,y) = 1
				continue
			}

			// Default: Empty accessible
			cell.VState = CellAccessible // v(x,y) = 0
		}
	}
}

// UpdateLocalVState updates virtual state after a move at position (x,y)
func UpdateLocalVState(b *Board, x, y int) {
	cell := &b.Cells[y][x]

	// Block all empty neighboring cells (Rule 1)
	for q := -1; q <= 1; q++ {
		for p := -1; p <= 1; p++ {
			nx, ny := x+p, y+q
			if nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
				neighborCell := &b.Cells[ny][nx]
				if neighborCell.Value == 0 {
					neighborCell.VState = CellBlocked // v = 1
				}
			}
		}
	}

	// Set the placed cell's virtual state (Rules 2 & 3)
	if cell.Value == 9 {
		cell.VState = CellAccessible // v(x,y) = 0 (permanent)
	} else {
		cell.VState = CellReplaceable // v(x,y) = 2
	}
}

// hasFilledNeighbor checks if a cell has any filled neighbors
func hasFilledNeighbor(b *Board, x, y int) bool {
	for q := -1; q <= 1; q++ {
		for p := -1; p <= 1; p++ {
			if p == 0 && q == 0 {
				continue // Skip the cell itself
			}
			nx, ny := x+p, y+q
			if nx >= 0 && nx < b.Size && ny >= 0 && ny < b.Size {
				if b.Cells[ny][nx].Value != 0 {
					return true
				}
			}
		}
	}
	return false
}

func FindBestBotMove(b *Board, botID string, hand []int, cfg *config.Config) (*Move, error) {
	moves := GenerateLegalMoves(b, hand, botID) // Add botID parameter

	if len(moves) == 0 {
		return nil, errors.New("no legal moves available")
	}

	var bestMove *Move
	bestScore := -1

	for _, m := range moves {
		score := EvaluateMove(b, m.X, m.Y, m.Card, botID, cfg)
		if score > bestScore {
			bestScore = score
			bestMove = &m
		}
	}

	return bestMove, nil
}
