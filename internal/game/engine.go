package game

import "slices"

func ApplyMove(b *Board, x, y int, owner string, card int) {
	cell := &b.Cells[y][x]
	cell.OwnerID = owner
	cell.Value = card
}

func GenerateLegalMoves(b Board, hand []int, playerID string) []Move {
	var moves []Move
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			cs := b.Cells[y][x]
			switch cs.VState {
			case CellPlaceable:
				for _, c := range hand {
					moves = append(moves, Move{X: x, Y: y, Card: c, PlayerID: playerID})
				}
			case CellReplaceable:
				if cs.OwnerID != "" && cs.OwnerID != playerID {
					for _, c := range hand {
						if c > cs.Value {
							moves = append(moves, Move{X: x, Y: y, Card: c, PlayerID: playerID})
						}
					}
				}
			}
		}
	}
	slices.SortFunc(moves, func(a, b Move) int {
		if a.Card == b.Card {
			if a.Y == b.Y {
				return a.X - b.X
			}
			return a.Y - b.Y
		}
		return a.Card - b.Card
	})
	return moves
}

func UpdateVState(b *Board) {
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			b.Cells[y][x].VState = CellBlocked
		}
	}
	center := b.Size / 2
	any := false
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			if b.Cells[y][x].OwnerID != "" {
				any = true
				break
			}
		}
		if any {
			break
		}
	}
	if !any {
		b.Cells[center][center].VState = CellPlaceable
		return
	}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for y := 0; y < b.Size; y++ {
		for x := 0; x < b.Size; x++ {
			cell := b.Cells[y][x]
			if cell.OwnerID == "" {
				for _, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if in(nx, ny, b.Size) && b.Cells[ny][nx].OwnerID != "" {
						b.Cells[y][x].VState = CellPlaceable
						break
					}
				}
			} else {
				b.Cells[y][x].VState = CellReplaceable
			}
		}
	}
}
