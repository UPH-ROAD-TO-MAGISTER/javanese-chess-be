package game

func IsWinningAfter(b Board, x, y int, owner string, card int) bool {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		count := 1
		i, j := x+d[0], y+d[1]
		for in(i, j, b.Size) && b.Cells[j][i].OwnerID == owner {
			count++
			i += d[0]
			j += d[1]
		}
		i, j = x-d[0], y-d[1]
		for in(i, j, b.Size) && b.Cells[j][i].OwnerID == owner {
			count++
			i -= d[0]
			j -= d[1]
		}
		if count >= 4 {
			return true
		}
	}
	return false
}
