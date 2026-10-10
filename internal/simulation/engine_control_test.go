package simulation

import (
	"errors"
	"testing"

	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/world"
)

func TestEnginePausesBeforeFirstStepAndResumes(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 2, Y: 0}

	engine, _ := newTestEngine(t, 4, 3, start, 100)

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	paused, err := engine.Pause()
	if err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	if paused.Status != StatusPaused {
		t.Fatalf("Status = %q, want %q", paused.Status, StatusPaused)
	}

	if paused.Rover.Status != rover.StatusPaused {
		t.Fatalf(
			"Rover.Status = %q, want %q",
			paused.Rover.Status,
			rover.StatusPaused,
		)
	}

	_, err = engine.Step()
	if !errors.Is(err, ErrMissionPaused) {
		t.Fatalf("Step() error = %v, want ErrMissionPaused", err)
	}

	afterRejectedStep := engine.Snapshot()
	if afterRejectedStep.Rover.Position != start {
		t.Errorf("rover moved while paused: %+v", afterRejectedStep.Rover.Position)
	}

	if afterRejectedStep.Steps != 0 {
		t.Errorf("Steps = %d while paused, want 0", afterRejectedStep.Steps)
	}

	resumed, err := engine.Resume()
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}

	if resumed.Status != StatusPlanned {
		t.Fatalf("Status = %q, want %q", resumed.Status, StatusPlanned)
	}

	firstStep, err := engine.Step()
	if err != nil {
		t.Fatalf("Step() after resume error: %v", err)
	}

	if firstStep.Rover.Position != (world.Point{X: 1, Y: 0}) {
		t.Errorf("Position = %+v, want (1, 0)", firstStep.Rover.Position)
	}

	if firstStep.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", firstStep.Status, StatusRunning)
	}
}

func TestEnginePausesAndResumesRunningMission(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 3, Y: 0}

	engine, _ := newTestEngine(t, 5, 3, start, 100)

	if err := engine.Plan(target); err != nil {
		t.Fatal(err)
	}

	firstStep, err := engine.Step()
	if err != nil {
		t.Fatalf("first Step() error: %v", err)
	}

	positionBeforePause := firstStep.Rover.Position
	batteryBeforePause := firstStep.Rover.BatteryPercent

	paused, err := engine.Pause()
	if err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	if paused.Status != StatusPaused {
		t.Fatalf("Status = %q, want %q", paused.Status, StatusPaused)
	}

	_, err = engine.Step()
	if !errors.Is(err, ErrMissionPaused) {
		t.Fatalf("Step() error = %v, want ErrMissionPaused", err)
	}

	duringPause := engine.Snapshot()

	if duringPause.Rover.Position != positionBeforePause {
		t.Errorf("rover moved while paused")
	}

	if duringPause.Rover.BatteryPercent != batteryBeforePause {
		t.Errorf("battery changed while paused")
	}

	resumed, err := engine.Resume()
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}

	if resumed.Status != StatusRunning {
		t.Fatalf("Status = %q, want %q", resumed.Status, StatusRunning)
	}

	secondStep, err := engine.Step()
	if err != nil {
		t.Fatalf("second Step() error: %v", err)
	}

	if secondStep.Rover.Position != (world.Point{X: 2, Y: 0}) {
		t.Errorf("Position = %+v, want (2, 0)", secondStep.Rover.Position)
	}

	if secondStep.Steps != 2 {
		t.Errorf("Steps = %d, want 2", secondStep.Steps)
	}
}

func TestEngineRejectsInvalidPauseAndResume(t *testing.T) {
	engine, _ := newTestEngine(
		t,
		3,
		3,
		world.Point{X: 0, Y: 0},
		100,
	)

	if _, err := engine.Pause(); err == nil {
		t.Fatal("Pause() should reject an idle engine")
	}

	if _, err := engine.Resume(); err == nil {
		t.Fatal("Resume() should reject an idle engine")
	}

	if err := engine.Plan(world.Point{X: 1, Y: 0}); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Resume(); err == nil {
		t.Fatal("Resume() should reject a mission that is not paused")
	}
}
