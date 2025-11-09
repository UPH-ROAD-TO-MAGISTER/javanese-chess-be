package game

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

	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			cell := b.Cells[y][x]

			// Condition 1: not blocked (VState == 1 means blocked)
			if cell.VState == CellBlocked {
				continue
			}

			// Skip permanent card 9 (VState == 0 and Value == 9)
			if cell.VState == CellAccessible && cell.Value == 9 {
				continue
			}

			for _, card := range hand {
				// Condition 2: card must be higher than current value
				if cell.Value >= card {
					continue
				}

				// Condition 3: cannot overwrite own card
				if cell.OwnerID == playerID {
					continue
				}

				moves = append(moves, Move{X: x, Y: y, Card: card, PlayerID: playerID})
			}
		}
	}

	return moves
}
