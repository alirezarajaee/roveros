package simulation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/alirezarajaee/roveros/internal/navigation"
	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/world"
)

func newTestEngine(
	t *testing.T,
	width, height int,
	start world.Point,
	battery float64,
) (*Engine, *world.Grid) {
	t.Helper()

	grid, err := world.NewGrid(width, height)
	if err != nil {
		t.Fatalf("NewGrid() error: %v", err)
	}

	initialRover, err := rover.New(start, battery)
	if err != nil {
		t.Fatalf("rover.New() error: %v", err)
	}

	engine, err := New(grid, initialRover)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return engine, grid
}

func TestEngineExecutesMissionToCompletion(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 2, Y: 1}

	engine, _ := newTestEngine(t, 5, 5, start, 100)

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	initial := engine.Snapshot()

	if initial.Status != StatusPlanned {
		t.Fatalf("Status = %q, want %q", initial.Status, StatusPlanned)
	}

	if len(initial.Path) != 4 {
		t.Fatalf("path has %d points, want 4", len(initial.Path))
	}

	if initial.Path[0] != start {
		t.Errorf("path starts at %+v, want %+v", initial.Path[0], start)
	}

	if initial.Path[len(initial.Path)-1] != target {
		t.Errorf("path does not end at target %+v", target)
	}

	requiredSteps := len(initial.Path) - 1

	for i := 0; i < requiredSteps; i++ {
		snapshot, err := engine.Step()
		if err != nil {
			t.Fatalf("Step() %d error: %v", i+1, err)
		}

		if i < requiredSteps-1 && snapshot.Status != StatusRunning {
			t.Fatalf(
				"Status after step %d = %q, want %q",
				i+1,
				snapshot.Status,
				StatusRunning,
			)
		}
	}

	final := engine.Snapshot()

	if final.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", final.Status, StatusCompleted)
	}

	if final.Rover.Position != target {
		t.Errorf("final position = %+v, want %+v", final.Rover.Position, target)
	}

	if final.Steps != requiredSteps {
		t.Errorf("Steps = %d, want %d", final.Steps, requiredSteps)
	}

	wantBattery := 100.0 - float64(requiredSteps)
	if final.Rover.BatteryPercent != wantBattery {
		t.Errorf(
			"BatteryPercent = %v, want %v",
			final.Rover.BatteryPercent,
			wantBattery,
		)
	}

	if final.Rover.Status != rover.StatusIdle {
		t.Errorf(
			"rover status = %q, want %q",
			final.Rover.Status,
			rover.StatusIdle,
		)
	}

	if _, err := engine.Step(); !errors.Is(err, ErrMissionCompleted) {
		t.Errorf("Step() after completion error = %v, want ErrMissionCompleted", err)
	}
}

func TestEngineRequiresMissionBeforeStepping(t *testing.T) {
	engine, _ := newTestEngine(
		t,
		3,
		3,
		world.Point{X: 0, Y: 0},
		100,
	)

	snapshot, err := engine.Step()

	if !errors.Is(err, ErrNoMission) {
		t.Fatalf("Step() error = %v, want ErrNoMission", err)
	}

	if snapshot.Status != StatusIdle {
		t.Errorf("Status = %q, want %q", snapshot.Status, StatusIdle)
	}
}

func TestEngineFailedPlanPreservesExistingMission(t *testing.T) {
	start := world.Point{X: 0, Y: 1}
	existingTarget := world.Point{X: 0, Y: 2}

	engine, grid := newTestEngine(t, 3, 3, start, 100)

	if err := engine.Plan(existingTarget); err != nil {
		t.Fatalf("initial Plan() error: %v", err)
	}

	before := engine.Snapshot()

	// Create a wall that separates the rover from the new target.
	for y := 0; y < 3; y++ {
		if err := grid.SetBlocked(world.Point{X: 1, Y: y}, true); err != nil {
			t.Fatal(err)
		}
	}

	err := engine.Plan(world.Point{X: 2, Y: 1})

	if !errors.Is(err, navigation.ErrNoPath) {
		t.Fatalf("Plan() error = %v, want ErrNoPath", err)
	}

	after := engine.Snapshot()

	if after.Status != before.Status {
		t.Errorf("Status changed from %q to %q", before.Status, after.Status)
	}

	if after.Target != before.Target {
		t.Errorf("Target changed from %+v to %+v", before.Target, after.Target)
	}

	if !reflect.DeepEqual(after.Path, before.Path) {
		t.Errorf("path changed after failed planning")
	}

	// The original route remains usable.
	snapshot, err := engine.Step()
	if err != nil {
		t.Fatalf("Step() on original mission error: %v", err)
	}

	if snapshot.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", snapshot.Status, StatusCompleted)
	}

	if snapshot.Rover.Position != existingTarget {
		t.Errorf("Position = %+v, want %+v", snapshot.Rover.Position, existingTarget)
	}
}

func TestEngineFailsSafelyWhenBatteryIsInsufficient(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 1, Y: 0}

	engine, _ := newTestEngine(t, 3, 3, start, 0.5)

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	_, err := engine.Step()
	if err == nil {
		t.Fatal("Step() expected an error due to insufficient battery")
	}

	snapshot := engine.Snapshot()

	if snapshot.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", snapshot.Status, StatusFailed)
	}

	if snapshot.Rover.Position != start {
		t.Errorf("Position changed to %+v after failed movement", snapshot.Rover.Position)
	}

	if snapshot.Rover.BatteryPercent != 0.5 {
		t.Errorf("BatteryPercent = %v, want 0.5", snapshot.Rover.BatteryPercent)
	}

	if snapshot.LastError == "" {
		t.Error("LastError should describe the failure")
	}

	_, err = engine.Step()
	if !errors.Is(err, ErrMissionFailed) {
		t.Errorf("Step() after failure error = %v, want ErrMissionFailed", err)
	}
}

func TestEngineCompletesMissionWhenAlreadyAtTarget(t *testing.T) {
	start := world.Point{X: 1, Y: 1}

	engine, _ := newTestEngine(t, 3, 3, start, 100)

	if err := engine.Plan(start); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	snapshot := engine.Snapshot()

	if snapshot.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", snapshot.Status, StatusCompleted)
	}

	if len(snapshot.Path) != 1 || snapshot.Path[0] != start {
		t.Errorf("Path = %+v, want only the starting point", snapshot.Path)
	}

	if snapshot.Steps != 0 {
		t.Errorf("Steps = %d, want 0", snapshot.Steps)
	}
}
