package world

// Point represents a cell coordinate on the 2D terrain grid.
// X increases to the right, and Y increases downward.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}