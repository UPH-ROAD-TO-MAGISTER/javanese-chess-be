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

	// Base heuristic values from the research table

	// Legal move base value
	DefaultLegalMoveValue = 30

	// Winning move (4-in-a-row)
	DefaultWWin = 10000

	// Threat detection (3 opponent cards in a row)
	DefaultWThreat = 200

	// Replace opponent's card values (context-dependent)
	DefaultReplaceWhenThreat = 200 // When blocking immediate threat
	DefaultReplacePotential  = 125 // When blocking potential threat

	// Position bonuses when replacing opponent's card
	DefaultReplacePosCenter = 75 // Center position in opponent's line
	DefaultReplacePosSide   = 50 // Side position in opponent's line

	// Block opponent's path values (context-dependent)
	DefaultBlockWhenThreat = 100 // Blocking 3-in-a-row completion
	DefaultBlockPotential  = 70  // Blocking 2-in-a-row extension

	// Formation building (our cards in a row)
	DefaultBuildAlignment2 = 50  // 2 of our cards in a row
	DefaultBuildAlignment3 = 100 // 3 of our cards in a row

	// Card management bonuses
	DefaultPlaySmallestCard = 60 // Bonus for playing smallest card in hand
	DefaultKeepNearCard     = 60 // Bonus for placing card close to our own cards
)

// Config holds all configuration values
type Config struct {
	HTTPAddr  string
	BoardSize int

	// Default heuristic weights (global)
	DefaultWeights HeuristicWeights
}

// HeuristicWeights represents AI evaluation parameters
type HeuristicWeights struct {
	// Base legal move value
	LegalMove int `json:"legal_move"`

	// Winning move (4-in-a-row)
	WWin int `json:"w_win"`

	// Threat detection (3 opponent cards in a row)
	WThreat int `json:"w_threat"`

	// Card values when blocking threat (high cards preferred: 1→20, 9→100)
	ReplaceValuesThreat map[int]int `json:"replace_values_threat"`

	// Card values for defensive play (low cards preferred: 1→100, 9→20)
	ReplaceValuesPotential map[int]int `json:"replace_values_potential"`

	// Replace opponent's card values (context-dependent)
	ReplaceWhenThreat int `json:"replace_when_threat"` // 200 when blocking immediate threat
	ReplacePotential  int `json:"replace_potential"`   // 125 when blocking potential threat

	// Position bonuses when replacing opponent's card
	ReplacePosCenter int `json:"replace_pos_center"` // 75 for center position
	ReplacePosSide   int `json:"replace_pos_side"`   // 50 for side position

	// Block opponent's path values (context-dependent)
	BlockWhenThreat int `json:"block_when_threat"` // 100 for blocking 3-in-a-row
	BlockPotential  int `json:"block_potential"`   // 70 for blocking 2-in-a-row

	// Formation building (our cards in a row)
	BuildAlignment2 int `json:"build_alignment_2"` // 50 for 2-in-a-row
	BuildAlignment3 int `json:"build_alignment_3"` // 100 for 3-in-a-row

	// Card management bonuses
	PlaySmallestCard int `json:"play_smallest_card"` // 60 for playing smallest card
	KeepNearCard     int `json:"keep_near_card"`     // 60 for placing near own cards
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
				// Base values from heuristic table
				LegalMove: DefaultLegalMoveValue, // 30
				WWin:      DefaultWWin,           // 10000
				WThreat:   DefaultWThreat,        // 200

				// Card values when blocking threat (high cards preferred: 1→20, 9→100)
				ReplaceValuesThreat: map[int]int{
					1: 20, 2: 30, 3: 40, 4: 50, 5: 60,
					6: 70, 7: 80, 8: 90, 9: 100,
				},

				// Card values for defensive play (low cards preferred: 1→100, 9→20)
				ReplaceValuesPotential: map[int]int{
					1: 100, 2: 90, 3: 80, 4: 70, 5: 60,
					6: 50, 7: 40, 8: 30, 9: 20,
				},

				// Replace opponent's card values
				ReplaceWhenThreat: DefaultReplaceWhenThreat, // 200
				ReplacePotential:  DefaultReplacePotential,  // 125

				// Position bonuses when replacing
				ReplacePosCenter: DefaultReplacePosCenter, // 75
				ReplacePosSide:   DefaultReplacePosSide,   // 50

				// Block opponent's path values
				BlockWhenThreat: DefaultBlockWhenThreat, // 100
				BlockPotential:  DefaultBlockPotential,  // 70

				// Formation building
				BuildAlignment2: DefaultBuildAlignment2, // 50
				BuildAlignment3: DefaultBuildAlignment3, // 100

				// Card management bonuses
				PlaySmallestCard: DefaultPlaySmallestCard, // 60
				KeepNearCard:     DefaultKeepNearCard,     // 60
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
	if w.LegalMove < 0 || w.WWin < 0 || w.WThreat < 0 ||
		w.ReplaceWhenThreat < 0 || w.ReplacePotential < 0 ||
		w.ReplacePosCenter < 0 || w.ReplacePosSide < 0 ||
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
