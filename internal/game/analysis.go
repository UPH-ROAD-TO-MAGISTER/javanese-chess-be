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

func chainLenAfter(b Board, x, y int, owner string) int {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	max := 1
	for _, d := range dirs {
		cnt := 1
		i, j := x+d[0], y+d[1]
		for in(i, j, b.Size) && b.Cells[j][i].OwnerID == owner {
			cnt++
			i += d[0]
			j += d[1]
		}
		i, j = x-d[0], y-d[1]
		for in(i, j, b.Size) && b.Cells[j][i].OwnerID == owner {
			cnt++
			i -= d[0]
			j -= d[1]
		}
		if cnt > max {
			max = cnt
		}
	}
	return max
}

func blocksImmediateThreat(b Board, x, y int, owner string) bool {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		for offset := -3; offset <= 0; offset++ {
			enemyCount := 0
			selfCount := 0
			emptyCount := 0
			valid := true
			for i := 0; i < 4; i++ {
				px := x + d[0]*(offset+i)
				py := y + d[1]*(offset+i)
				if !in(px, py, b.Size) {
					valid = false
					break
				}
				cell := b.Cells[py][px]
				if px == x && py == y {
					selfCount++
					continue
				}
				switch cell.OwnerID {
				case "":
					emptyCount++
				case owner:
					selfCount++
				default:
					enemyCount++
				}
			}
			if !valid {
				continue
			}
			if enemyCount == 3 && selfCount == 1 && emptyCount == 0 {
				return true
			}
		}
	}
	return false
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

			// Condition 1: not blocked
			if cell.VState == 1 {
				continue
			}

			// Skip permanent card 9
			if cell.VState == 0 && cell.Value == 9 {
				continue
			}

			for _, card := range hand {
				// Condition 2: card must be higher
				if cell.Value >= card {
					continue
				}

				// Condition 3: cannot overwrite own card
				if cell.OwnerID == playerID {
					continue
				}

				moves = append(moves, Move{X: x, Y: y, Card: card})
			}
		}
	}

	return moves
}
