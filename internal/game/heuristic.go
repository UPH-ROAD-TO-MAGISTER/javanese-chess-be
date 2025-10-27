package game

import "javanese-chess/internal/config"

func HeuristicScore(b Board, m Move, cfg config.Config) int {
	w := cfg.Weights
	score := 0

	// Immediate win
	if IsWinningAfter(b, m.X, m.Y, m.PlayerID, m.Card) {
		return w.WWin
	}

	// Threat blocking
	if blocksImmediateThreat(b, m.X, m.Y, m.PlayerID) {
		score += w.WThreat
	}

	// Build 2/3 chains
	cl := chainLenAfter(b, m.X, m.Y, m.PlayerID)
	if cl >= 3 {
		score += 2 * w.WBuild
	} else if cl == 2 {
		score += w.WBuild
	}

	// Overwrite enemy
	if b.Cells[m.Y][m.X].OwnerID != "" && b.Cells[m.Y][m.X].OwnerID != m.PlayerID && m.Card > b.Cells[m.Y][m.X].Value {
		score += w.WOverwrite
	}

	// Card policy: higher if blocking, else conserve high cards
	if blocksImmediateThreat(b, m.X, m.Y, m.PlayerID) {
		score += m.Card * w.WCardVal
	} else {
		score += (10 - m.Card) * w.WCardVal
	}

	return score
}
