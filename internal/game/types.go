package game

type CellState int

const (
	CellBlocked CellState = iota
	CellPlaceable
	CellReplaceable
)

type Cell struct {
	OwnerID string    `json:"ownerId,omitempty"`
	Value   int       `json:"value,omitempty"`
	VState  CellState `json:"vstate"`
}

type Board struct {
	Size  int      `json:"size"`
	Cells [][]Cell `json:"cells"`
}

func NewBoard(size int) Board {
	c := make([][]Cell, size)
	for i := range c {
		c[i] = make([]Cell, size)
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
