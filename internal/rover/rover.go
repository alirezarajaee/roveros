package rover

import (
	"fmt"
	"math"

	"github.com/alirezarajaee/roveros/internal/world"
)

// Status represents the current operational state of the rover.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusMoving   Status = "moving"
	StatusPaused   Status = "paused"
	StatusDisabled Status = "disabled"
)

// Rover represents the state of a simulated Mars rover.
type Rover struct {
	Position       world.Point `json:"position"`
	BatteryPercent float64     `json:"battery_percent"`
	Status         Status      `json:"status"`
}

// New creates a rover with the given position and battery level.
func New(position world.Point, batteryPercent float64) (Rover, error) {
	r := Rover{
		Position:       position,
		BatteryPercent: batteryPercent,
		Status:         StatusIdle,
	}

	if err := r.Validate(); err != nil {
		return Rover{}, err
	}

	return r, nil
}

// Validate checks whether the rover state contains valid values.
func (r Rover) Validate() error {
	if math.IsNaN(r.BatteryPercent) ||
		math.IsInf(r.BatteryPercent, 0) ||
		r.BatteryPercent < 0 ||
		r.BatteryPercent > 100 {
		return fmt.Errorf(
			"battery percentage must be between 0 and 100",
		)
	}

	switch r.Status {
	case StatusIdle, StatusMoving, StatusPaused, StatusDisabled:
		return nil
	default:
		return fmt.Errorf("invalid rover status: %q", r.Status)
	}
}
