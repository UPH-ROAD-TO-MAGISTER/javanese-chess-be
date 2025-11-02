package config

import (
	"os"
	"reflect"
	"sync"
)

// Constants from the research paper "The Mechanics and Heuristics of Javanese Chess" Section 2.4
// These values are based on empirical research and should not be changed without proper analysis
const (
	// Game Constants
	DefaultBoardSize = 9 // Standard Javanese Chess board is 9x9

	// Heuristic Weight Constants - H(s,a) = W₁·f_win + W₂·f_threat + W₃·f_replace + W₄·f_block + W₅·f_align + W₆·f_cost

	// DefaultWWin - W₁: Immediate winning move detection (4-in-a-row)
	// Highest priority - must always take a winning move
	DefaultWWin = 10000

	// DefaultWThreat - W₂: Blocking opponent's immediate threat (3-in-a-row)
	// Second priority - prevent opponent from winning
	DefaultWThreat = 200

	// DefaultWReplaceValue - W₃: Overwriting opponent cards (strategic replacement)
	// Medium-high priority - gain positional advantage
	DefaultWReplaceValue = 125

	// DefaultWBlockPath - W₄: Blocking enemy paths (cutting opponent lines)
	// Medium priority - disrupt opponent strategy
	DefaultWBlockPath = 70

	// DefaultWBuildAlignment - W₅: Building own alignments (2 or 3 in a row)
	// Medium priority - prepare future winning positions
	DefaultWBuildAlignment = 50

	// DefaultWCardCost - W₆: Card value management (resource efficiency)
	// Lowest priority - tie-breaker for equal positions
	DefaultWCardCost = 1

	// Additional heuristic values from the provided table
	DefaultLegalMoveValue    = 30
	DefaultReplaceWhenThreat = 200
	DefaultReplacePotential  = 125
	DefaultReplacePosMiddle  = 75
	DefaultReplacePosSide    = 50
	DefaultBlockWhenThreat   = 100
	DefaultBlockPotential    = 70
	DefaultBuildAlignment2   = 50
	DefaultBuildAlignment3   = 100
	DefaultPlaySmallestCard  = 60
	DefaultKeepNearCard      = 60
)

// Config holds all configuration values
type Config struct {
	HTTPAddr  string
	BoardSize int

	// Default heuristic weights (global)
	DefaultWeights HeuristicWeights
}

// HeuristicWeights represents AI evaluation parameters based on Section 2.4
// H(s,a) = W₁·f_win + W₂·f_threat_block + W₃·f_replace_value + W₄·f_block_path + W₅·f_build_alignment + W₆·f_card_cost
type HeuristicWeights struct {
	// W₁: Winning move detection (4-in-a-row)
	WWin int `json:"w_win"`

	// W₂: Blocking opponent's immediate threat (3-in-a-row)
	WThreat int `json:"w_threat"`

	// W₃: Overwriting opponent cards (strategic replacement)
	WReplaceValue int `json:"w_replace_value"`

	// W₄: Blocking enemy paths (cutting opponent lines)
	WBlockPath int `json:"w_block_path"`

	// W₅: Building own alignments (2 or 3 in a row)
	WBuildAlignment int `json:"w_build_alignment"`

	// W₆: Card value management (resource efficiency)
	WCardCost int `json:"w_card_cost"`

	// Legal move value
	LegalMove int `json:"legal_move"`

	// Replace values when blocking an immediate threat (card 1..9 -> indices 0..8)
	ReplaceValuesThreat map[int]int `json:"replace_values_threat"`

	// Replace values for potential threats (prioritize small cards: card1..9)
	ReplaceValuesPotential map[int]int `json:"replace_values_potential"`

	// Replace weights (contextual)
	ReplaceWhenThreat int `json:"replace_when_threat"`
	ReplacePotential  int `json:"replace_potential"`

	// Position modifiers for replacement
	ReplacePosMiddle int `json:"replace_pos_middle"`
	ReplacePosSide   int `json:"replace_pos_side"`

	// Blocking weights
	BlockWhenThreat int `json:"block_when_threat"`
	BlockPotential  int `json:"block_potential"`

	// Build alignment specific weights
	BuildAlignment2 int `json:"build_alignment_2"`
	BuildAlignment3 int `json:"build_alignment_3"`

	// Smallest-card and proximity preferences
	PlaySmallestCard int `json:"play_smallest_card"`
	KeepNearCard     int `json:"keep_near_card"`
}

// RoomConfig holds configuration for a specific room
type RoomConfig struct {
	RoomCode string           `json:"room_code"`
	Weights  HeuristicWeights `json:"weights"`
	mu       sync.RWMutex
}

var globalConfig *Config
var once sync.Once

// Load initializes the global configuration with default values from paper
func Load() *Config {
	once.Do(func() {
		globalConfig = &Config{
			HTTPAddr:  getHTTPAddr(),
			BoardSize: DefaultBoardSize,
			DefaultWeights: HeuristicWeights{
				// Values from research paper Section 2.4
				WWin:            DefaultWWin,
				WThreat:         DefaultWThreat,
				WReplaceValue:   DefaultWReplaceValue,
				WBlockPath:      DefaultWBlockPath,
				WBuildAlignment: DefaultWBuildAlignment,
				WCardCost:       DefaultWCardCost, // Additional defaults per provided heuristic table
				LegalMove:       DefaultLegalMoveValue,

				// Replace when immediate threat (card values 1..9)
				ReplaceValuesThreat: map[int]int{1: 20, 2: 30, 3: 40, 4: 50, 5: 60, 6: 70, 7: 80, 8: 90, 9: 100},

				// Replace values for potential (prefer small cards: 1..9 -> 100..20)
				ReplaceValuesPotential: map[int]int{1: 100, 2: 90, 3: 80, 4: 70, 5: 60, 6: 50, 7: 40, 8: 30, 9: 20},

				ReplaceWhenThreat: DefaultReplaceWhenThreat,
				ReplacePotential:  DefaultReplacePotential,

				ReplacePosMiddle: DefaultReplacePosMiddle,
				ReplacePosSide:   DefaultReplacePosSide,

				BlockWhenThreat: DefaultBlockWhenThreat,
				BlockPotential:  DefaultBlockPotential,

				BuildAlignment2: DefaultBuildAlignment2,
				BuildAlignment3: DefaultBuildAlignment3,

				PlaySmallestCard: DefaultPlaySmallestCard,
				KeepNearCard:     DefaultKeepNearCard,
			},
		}
	})
	return globalConfig
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		return Load()
	}
	return globalConfig
}

// NewRoomConfig creates a new room configuration with default weights
func NewRoomConfig(roomCode string) *RoomConfig {
	return &RoomConfig{
		RoomCode: roomCode,
		Weights:  Get().DefaultWeights,
	}
}

// GetWeights returns the current weights for this room (thread-safe)
func (rc *RoomConfig) GetWeights() HeuristicWeights {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.Weights
}

// SetWeights updates the weights for this room (thread-safe)
func (rc *RoomConfig) SetWeights(weights HeuristicWeights) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.Weights = weights
}

// IsCustomized checks if weights differ from defaults
func (rc *RoomConfig) IsCustomized() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	defaults := Get().DefaultWeights
	return !reflect.DeepEqual(rc.Weights, defaults)
}

// ValidateWeights checks if weights are within reasonable ranges
func (w *HeuristicWeights) ValidateWeights() bool {
	// All weights should be non-negative
	if w.WWin < 0 || w.WThreat < 0 || w.WReplaceValue < 0 || w.WBlockPath < 0 ||
		w.WBuildAlignment < 0 || w.WCardCost < 0 || w.LegalMove < 0 ||
		w.ReplaceWhenThreat < 0 || w.ReplacePotential < 0 ||
		w.ReplacePosMiddle < 0 || w.ReplacePosSide < 0 ||
		w.BlockWhenThreat < 0 || w.BlockPotential < 0 ||
		w.BuildAlignment2 < 0 || w.BuildAlignment3 < 0 ||
		w.PlaySmallestCard < 0 || w.KeepNearCard < 0 {
		return false
	}
	for _, v := range w.ReplaceValuesThreat {
		if v < 0 {
			return false
		}
	}
	for _, v := range w.ReplaceValuesPotential {
		if v < 0 {
			return false
		}
	}
	return true
}

// getHTTPAddr returns the HTTP address from environment or default
// This is kept configurable for deployment flexibility (dev/staging/prod)
func getHTTPAddr() string {
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	return ":9000" // Default port
}

// DefaultPlayerColors defines the available colors for players
var DefaultPlayerColors = []string{"red", "green", "blue", "purple"}
