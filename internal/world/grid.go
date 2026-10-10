package world

import "fmt"

// Grid represents a two-dimensional traversable terrain.
type Grid struct {
	width   int
	height  int
	blocked map[Point]struct{}
}

// NewGrid creates an empty grid with positive dimensions.
func NewGrid(width, height int) (*Grid, error) {
	if width <= 0 {
		return nil, fmt.Errorf("grid width must be positive")
	}

	if height <= 0 {
		return nil, fmt.Errorf("grid height must be positive")
	}

	return &Grid{
		width:   width,
		height:  height,
		blocked: make(map[Point]struct{}),
	}, nil
}

// Width returns the number of columns in the grid.
func (g *Grid) Width() int {
	return g.width
}

// Height returns the number of rows in the grid.
func (g *Grid) Height() int {
	return g.height
}

// InBounds reports whether a point is inside the grid.
func (g *Grid) InBounds(p Point) bool {
	return g != nil &&
		p.X >= 0 &&
		p.Y >= 0 &&
		p.X < g.width &&
		p.Y < g.height
}

// IsWalkable reports whether a rover can occupy a point.
func (g *Grid) IsWalkable(p Point) bool {
	if !g.InBounds(p) {
		return false
	}

	_, isBlocked := g.blocked[p]
	return !isBlocked
}

// SetBlocked marks a cell as blocked or makes it traversable again.
func (g *Grid) SetBlocked(p Point, blocked bool) error {
	if !g.InBounds(p) {
		return fmt.Errorf(
			"point (%d, %d) is outside the grid",
			p.X,
			p.Y,
		)
	}

	if blocked {
		g.blocked[p] = struct{}{}
	} else {
		delete(g.blocked, p)
	}

	return nil
}
