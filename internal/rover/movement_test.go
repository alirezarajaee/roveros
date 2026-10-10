package rover

import (
	"testing"

	"github.com/alirezarajaee/roveros/internal/world"
)

func newMovementFixture(t *testing.T, battery float64) (*Rover, *world.Grid) {
	t.Helper()

	grid, err := world.NewGrid(5, 5)
	if err != nil {
		t.Fatalf("NewGrid() error: %v", err)
	}

	r, err := New(world.Point{X: 2, Y: 2}, battery)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return &r, grid
}

func TestMoveToMovesOneCellAndConsumesEnergy(t *testing.T) {
	tests := []struct {
		name        string
		destination world.Point
	}{
		{name: "up", destination: world.Point{X: 2, Y: 1}},
		{name: "down", destination: world.Point{X: 2, Y: 3}},
		{name: "left", destination: world.Point{X: 1, Y: 2}},
		{name: "right", destination: world.Point{X: 3, Y: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, grid := newMovementFixture(t, 50)

			if err := r.MoveTo(grid, tt.destination); err != nil {
				t.Fatalf("MoveTo() error: %v", err)
			}

			if r.Position != tt.destination {
				t.Errorf("Position = %+v, want %+v", r.Position, tt.destination)
			}

			if r.BatteryPercent != 49 {
				t.Errorf("BatteryPercent = %v, want 49", r.BatteryPercent)
			}

			if r.Status != StatusMoving {
				t.Errorf("Status = %q, want %q", r.Status, StatusMoving)
			}
		})
	}
}

func TestMoveToRejectsInvalidDestinationsWithoutChangingState(t *testing.T) {
	tests := []struct {
		name        string
		destination world.Point
		block       bool
	}{
		{
			name:        "same position",
			destination: world.Point{X: 2, Y: 2},
		},
		{
			name:        "diagonal move",
			destination: world.Point{X: 3, Y: 3},
		},
		{
			name:        "multiple cells",
			destination: world.Point{X: 4, Y: 2},
		},
		{
			name:        "outside grid",
			destination: world.Point{X: 5, Y: 2},
		},
		{
			name:        "blocked cell",
			destination: world.Point{X: 3, Y: 2},
			block:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, grid := newMovementFixture(t, 50)
			initialPosition := r.Position
			initialBattery := r.BatteryPercent
			initialStatus := r.Status

			if tt.block {
				if err := grid.SetBlocked(tt.destination, true); err != nil {
					t.Fatalf("SetBlocked() error: %v", err)
				}
			}

			if err := r.MoveTo(grid, tt.destination); err == nil {
				t.Fatal("MoveTo() expected an error")
			}

			if r.Position != initialPosition {
				t.Errorf("Position changed after failed move: %+v", r.Position)
			}

			if r.BatteryPercent != initialBattery {
				t.Errorf("Battery changed after failed move: %v", r.BatteryPercent)
			}

			if r.Status != initialStatus {
				t.Errorf("Status changed after failed move: %q", r.Status)
			}
		})
	}
}

func TestMoveToRejectsInsufficientBattery(t *testing.T) {
	r, grid := newMovementFixture(t, 0.5)
	initialPosition := r.Position
	initialBattery := r.BatteryPercent

	err := r.MoveTo(grid, world.Point{X: 3, Y: 2})
	if err == nil {
		t.Fatal("MoveTo() expected an error when battery is insufficient")
	}

	if r.Position != initialPosition {
		t.Errorf("Position changed after failed move: %+v", r.Position)
	}

	if r.BatteryPercent != initialBattery {
		t.Errorf("Battery changed after failed move: %v", r.BatteryPercent)
	}
}

func TestMoveToRejectsPausedAndDisabledRover(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{name: "paused", status: StatusPaused},
		{name: "disabled", status: StatusDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, grid := newMovementFixture(t, 50)
			r.Status = tt.status

			initialPosition := r.Position
			initialBattery := r.BatteryPercent

			err := r.MoveTo(grid, world.Point{X: 3, Y: 2})
			if err == nil {
				t.Fatal("MoveTo() expected an error for this status")
			}

			if r.Position != initialPosition {
				t.Errorf("Position changed after rejected move")
			}

			if r.BatteryPercent != initialBattery {
				t.Errorf("Battery changed after rejected move")
			}

			if r.Status != tt.status {
				t.Errorf("Status = %q, want %q", r.Status, tt.status)
			}
		})
	}
}
