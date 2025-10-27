package config

import (
	"os"
	"strconv"
)

type Weights struct {
	WWin       int
	WThreat    int
	WOverwrite int
	WBlock     int
	WBuild     int
	WCardVal   int

	BonusThreatMid      int
	BonusThreatEdge     int
	BonusSmallestInHand int

	BoardSize int
}

type Config struct {
	Weights Weights
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func Load() Config {
	return Config{
		Weights: Weights{
			WWin:                getenvInt("W_WIN", 10000),
			WThreat:             getenvInt("W_THREAT", 200),
			WOverwrite:          getenvInt("W_OVERWRITE", 125),
			WBlock:              getenvInt("W_BLOCK", 70),
			WBuild:              getenvInt("W_BUILD", 60),
			WCardVal:            getenvInt("W_CARDVAL", 1),
			BonusThreatMid:      getenvInt("BONUS_THREAT_MID", 75),
			BonusThreatEdge:     getenvInt("BONUS_THREAT_EDGE", 50),
			BonusSmallestInHand: getenvInt("BONUS_SMALLEST", 60),
			BoardSize:           getenvInt("BOARD_SIZE", 9),
		},
	}
}
