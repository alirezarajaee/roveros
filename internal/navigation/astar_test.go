package navigation

import (
	"errors"
	"testing"

	"github.com/alirezarajaee/roveros/internal/world"
)

func newNavigationGrid(t *testing.T, width, height int) *world.Grid {
	t.Helper()

	grid, err := world.NewGrid(width, height)
	if err != nil {
		t.Fatalf("NewGrid() error: %v", err)
	}

	return grid
}

func TestFindPathReturnsShortestPathOnOpenGrid(t *testing.T) {
	grid := newNavigationGrid(t, 5, 5)

	start := world.Point{X: 0, Y: 0}
	goal := world.Point{X: 4, Y: 3}

	path, err := FindPath(grid, start, goal)
	if err != nil {
		t.Fatalf("FindPath() error: %v", err)
	}

	// Manhattan distance is 7, so a shortest path contains 8 points.
	if len(path) != 8 {
		t.Fatalf("path length = %d, want 8 points", len(path))
	}

	if path[0] != start {
		t.Errorf("path starts at %+v, want %+v", path[0], start)
	}

	if path[len(path)-1] != goal {
		t.Errorf("path ends at %+v, want %+v", path[len(path)-1], goal)
	}

	assertValidPath(t, grid, path)
}

func TestFindPathRoutesAroundObstacleWall(t *testing.T) {
	grid := newNavigationGrid(t, 5, 5)

	// This wall forces the rover to travel around its lower end.
	for y := 0; y < 4; y++ {
		if err := grid.SetBlocked(world.Point{X: 2, Y: y}, true); err != nil {
			t.Fatal(err)
		}
	}

	start := world.Point{X: 1, Y: 2}
	goal := world.Point{X: 3, Y: 2}

	path, err := FindPath(grid, start, goal)
	if err != nil {
		t.Fatalf("FindPath() error: %v", err)
	}

	// The shortest detour has 6 moves, or 7 points.
	if len(path) != 7 {
		t.Fatalf("path length = %d, want 7 points", len(path))
	}

	assertValidPath(t, grid, path)

	for _, point := range path {
		if point.X == 2 && point.Y < 4 {
			t.Fatalf("path crosses blocked wall at %+v", point)
		}
	}
}

func TestFindPathWhenStartEqualsGoal(t *testing.T) {
	grid := newNavigationGrid(t, 3, 3)
	point := world.Point{X: 1, Y: 1}

	path, err := FindPath(grid, point, point)
	if err != nil {
		t.Fatalf("FindPath() error: %v", err)
	}

	if len(path) != 1 || path[0] != point {
		t.Fatalf("path = %+v, want [%+v]", path, point)
	}
}

func TestFindPathReturnsNoPathWhenDestinationIsUnreachable(t *testing.T) {
	grid := newNavigationGrid(t, 3, 3)

	// A complete vertical wall separates the start from the goal.
	for y := 0; y < 3; y++ {
		if err := grid.SetBlocked(world.Point{X: 1, Y: y}, true); err != nil {
			t.Fatal(err)
		}
	}

	start := world.Point{X: 0, Y: 1}
	goal := world.Point{X: 2, Y: 1}

	path, err := FindPath(grid, start, goal)

	if !errors.Is(err, ErrNoPath) {
		t.Fatalf("error = %v, want ErrNoPath", err)
	}

	if path != nil {
		t.Errorf("path = %+v, want nil", path)
	}
}

func TestFindPathRejectsInvalidEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		start     world.Point
		goal      world.Point
		blocked   world.Point
		blockCell bool
		wantError error
	}{
		{
			name:      "start outside grid",
			start:     world.Point{X: -1, Y: 0},
			goal:      world.Point{X: 2, Y: 2},
			wantError: ErrStartOutOfBounds,
		},
		{
			name:      "goal outside grid",
			start:     world.Point{X: 0, Y: 0},
			goal:      world.Point{X: 3, Y: 2},
			wantError: ErrGoalOutOfBounds,
		},
		{
			name:      "blocked start",
			start:     world.Point{X: 0, Y: 0},
			goal:      world.Point{X: 2, Y: 2},
			blocked:   world.Point{X: 0, Y: 0},
			blockCell: true,
			wantError: ErrStartBlocked,
		},
		{
			name:      "blocked goal",
			start:     world.Point{X: 0, Y: 0},
			goal:      world.Point{X: 2, Y: 2},
			blocked:   world.Point{X: 2, Y: 2},
			blockCell: true,
			wantError: ErrGoalBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := newNavigationGrid(t, 3, 3)

			if tt.blockCell {
				if err := grid.SetBlocked(tt.blocked, true); err != nil {
					t.Fatal(err)
				}
			}

			path, err := FindPath(grid, tt.start, tt.goal)

			if !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}

			if path != nil {
				t.Errorf("path = %+v, want nil", path)
			}
		})
	}
}

func TestFindPathRejectsNilGrid(t *testing.T) {
	path, err := FindPath(nil, world.Point{}, world.Point{X: 1, Y: 1})

	if !errors.Is(err, ErrNilGrid) {
		t.Fatalf("error = %v, want ErrNilGrid", err)
	}

	if path != nil {
		t.Errorf("path = %+v, want nil", path)
	}
}

func assertValidPath(t *testing.T, grid *world.Grid, path []world.Point) {
	t.Helper()

	if len(path) == 0 {
		t.Fatal("path must not be empty")
	}

	for i, point := range path {
		if !grid.IsWalkable(point) {
			t.Errorf("path contains non-walkable point %+v", point)
		}

		if i == 0 {
			continue
		}

		previous := path[i-1]
		distance := abs(point.X-previous.X) + abs(point.Y-previous.Y)

		if distance != 1 {
			t.Errorf(
				"invalid move between %+v and %+v",
				previous,
				point,
			)
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
