package world

import "testing"

func TestNewGrid(t *testing.T) {
	grid, err := NewGrid(10, 8)
	if err != nil {
		t.Fatalf("NewGrid() returned unexpected error: %v", err)
	}

	if grid.Width() != 10 {
		t.Errorf("Width() = %d, want 10", grid.Width())
	}

	if grid.Height() != 8 {
		t.Errorf("Height() = %d, want 8", grid.Height())
	}
}

func TestNewGridRejectsInvalidDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "zero width", width: 0, height: 5},
		{name: "negative width", width: -1, height: 5},
		{name: "zero height", width: 5, height: 0},
		{name: "negative height", width: 5, height: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGrid(tt.width, tt.height)
			if err == nil {
				t.Fatal("NewGrid() expected an error for invalid dimensions")
			}
		})
	}
}

func TestGridBounds(t *testing.T) {
	grid, err := NewGrid(10, 8)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{name: "top left", point: Point{X: 0, Y: 0}, want: true},
		{name: "bottom right", point: Point{X: 9, Y: 7}, want: true},
		{name: "negative x", point: Point{X: -1, Y: 0}, want: false},
		{name: "negative y", point: Point{X: 0, Y: -1}, want: false},
		{name: "x outside grid", point: Point{X: 10, Y: 0}, want: false},
		{name: "y outside grid", point: Point{X: 0, Y: 8}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grid.InBounds(tt.point)
			if got != tt.want {
				t.Errorf("InBounds(%+v) = %v, want %v", tt.point, got, tt.want)
			}
		})
	}
}

func TestGridBlocksAndUnblocksCells(t *testing.T) {
	grid, err := NewGrid(10, 8)
	if err != nil {
		t.Fatal(err)
	}

	point := Point{X: 3, Y: 4}

	if !grid.IsWalkable(point) {
		t.Fatal("new grid cell should be walkable")
	}

	if err := grid.SetBlocked(point, true); err != nil {
		t.Fatalf("SetBlocked() returned unexpected error: %v", err)
	}

	if grid.IsWalkable(point) {
		t.Fatal("blocked cell should not be walkable")
	}

	if err := grid.SetBlocked(point, false); err != nil {
		t.Fatalf("SetBlocked() returned unexpected error: %v", err)
	}

	if !grid.IsWalkable(point) {
		t.Fatal("unblocked cell should be walkable")
	}
}

func TestGridRejectsOutOfBoundsChanges(t *testing.T) {
	grid, err := NewGrid(10, 8)
	if err != nil {
		t.Fatal(err)
	}

	point := Point{X: 10, Y: 3}

	if err := grid.SetBlocked(point, true); err == nil {
		t.Fatal("SetBlocked() expected an error for an out-of-bounds point")
	}

	if grid.IsWalkable(point) {
		t.Fatal("out-of-bounds point must not be walkable")
	}
}
