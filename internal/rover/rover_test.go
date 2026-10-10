package rover

import (
	"math"
	"testing"

	"github.com/alirezarajaee/roveros/internal/world"
)

func TestNewAcceptsValidBatteryPercentages(t *testing.T) {
	tests := []struct {
		name    string
		battery float64
	}{
		{name: "zero", battery: 0},
		{name: "middle", battery: 50},
		{name: "full", battery: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(world.Point{X: 2, Y: 3}, tt.battery)
			if err != nil {
				t.Fatalf("New() returned unexpected error: %v", err)
			}

			if r.BatteryPercent != tt.battery {
				t.Errorf(
					"BatteryPercent = %v, want %v",
					r.BatteryPercent,
					tt.battery,
				)
			}

			if r.Status != StatusIdle {
				t.Errorf("Status = %q, want %q", r.Status, StatusIdle)
			}
		})
	}
}

func TestNewRejectsInvalidBatteryPercentages(t *testing.T) {
	tests := []struct {
		name    string
		battery float64
	}{
		{name: "negative", battery: -1},
		{name: "above maximum", battery: 101},
		{name: "not a number", battery: math.NaN()},
		{name: "positive infinity", battery: math.Inf(1)},
		{name: "negative infinity", battery: math.Inf(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(world.Point{}, tt.battery)
			if err == nil {
				t.Fatalf(
					"New() expected an error for battery %v",
					tt.battery,
				)
			}
		})
	}
}

func TestNewPreservesInitialPosition(t *testing.T) {
	initial := world.Point{X: 5, Y: 7}

	r, err := New(initial, 80)
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	if r.Position != initial {
		t.Errorf("Position = %+v, want %+v", r.Position, initial)
	}
}

func TestValidateRejectsInvalidStatus(t *testing.T) {
	r := Rover{
		Position:       world.Point{X: 1, Y: 1},
		BatteryPercent: 50,
		Status:         Status("unknown"),
	}

	if err := r.Validate(); err == nil {
		t.Fatal("Validate() expected an error for an invalid status")
	}
}
