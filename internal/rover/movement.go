package rover

import (
	"fmt"

	"github.com/alirezarajaee/roveros/internal/world"
)

// movementEnergyCost is the initial energy cost per cell, in percentage points.
const movementEnergyCost = 1.0

// MoveTo moves the rover exactly one cell in a cardinal direction.
func (r *Rover) MoveTo(grid *world.Grid, destination world.Point) error {
	if r == nil {
		return fmt.Errorf("rover is nil")
	}

	if grid == nil {
		return fmt.Errorf("grid is nil")
	}

	if err := r.Validate(); err != nil {
		return fmt.Errorf("invalid rover state: %w", err)
	}

	if r.Status == StatusPaused {
		return fmt.Errorf("rover is paused")
	}

	if r.Status == StatusDisabled {
		return fmt.Errorf("rover is disabled")
	}

	if !grid.IsWalkable(r.Position) {
		return fmt.Errorf("current rover position is not walkable")
	}

	dx := destination.X - r.Position.X
	dy := destination.Y - r.Position.Y

	isHorizontalMove := (dx == 1 || dx == -1) && dy == 0
	isVerticalMove := (dy == 1 || dy == -1) && dx == 0

	if !isHorizontalMove && !isVerticalMove {
		return fmt.Errorf("destination must be exactly one cell away")
	}

	if !grid.IsWalkable(destination) {
		return fmt.Errorf(
			"destination (%d, %d) is not walkable",
			destination.X,
			destination.Y,
		)
	}

	if r.BatteryPercent < movementEnergyCost {
		return fmt.Errorf("insufficient battery to move")
	}

	// Update state only after all movement checks have passed.
	r.Position = destination
	r.BatteryPercent -= movementEnergyCost
	r.Status = StatusMoving

	return nil
}
