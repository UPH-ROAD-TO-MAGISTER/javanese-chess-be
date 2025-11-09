package game

type CellVState int

const (
	CellAccessible  CellVState = 0 // Empty and accessible, or permanent (card 9)
	CellBlocked     CellVState = 1 // Empty but blocked (has filled neighbors)
	CellReplaceable CellVState = 2 // Filled and can be overwritten
)

type Cell struct {
	Value   int        `json:"value"`   // Card value (0 if empty)
	VState  CellVState `json:"vState"`  // State of the cell (e.g., placeable or not)
	OwnerID string     `json:"ownerId"` // ID of the player who owns the cell
}

type Board struct {
	Size  int      `json:"size"`
	Cells [][]Cell `json:"cells"`
}

func NewBoard(size int) Board {
	if size <= 0 {
		size = 9 // Default to 9x9 board
	}

	c := make([][]Cell, size)
	for i := range c {
		c[i] = make([]Cell, size)
		for j := range c[i] {
			c[i][j] = Cell{
				Value:  0,              // No card placed yet
				VState: CellAccessible, // Empty cells start as accessible (0)
			}
		}
	}

	return Board{
		Size:  size,
		Cells: c,
	}
}

type Move struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Card     int    `json:"value"`
	PlayerID string `json:"playerId"`
}
