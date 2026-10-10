package simulation

import (
	"errors"
	"testing"

	"github.com/alirezarajaee/roveros/internal/fault"
	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/world"
)

func TestSensorFailurePausesMissionUntilRecovery(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 3, Y: 0}

	engine, _ := newTestEngine(t, 5, 3, start, 100)

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	first, err := engine.Step()
	if err != nil {
		t.Fatalf("first Step() error: %v", err)
	}

	positionBeforeFault := first.Rover.Position
	batteryBeforeFault := first.Rover.BatteryPercent
	stepsBeforeFault := first.Steps

	event, err := engine.InjectSensorFailure(fault.ComponentNavigationSensor)
	if err != nil {
		t.Fatalf("InjectSensorFailure() error: %v", err)
	}

	if event.Status != fault.StatusActive {
		t.Errorf("fault status = %q, want %q", event.Status, fault.StatusActive)
	}

	paused := engine.Snapshot()

	if paused.Status != StatusPaused {
		t.Fatalf("mission status = %q, want %q", paused.Status, StatusPaused)
	}

	if paused.Rover.Status != rover.StatusPaused {
		t.Errorf("rover status = %q, want %q", paused.Rover.Status, rover.StatusPaused)
	}

	if paused.Rover.Position != positionBeforeFault {
		t.Errorf("position changed during fault injection")
	}

	if len(paused.ActiveFaults) != 1 {
		t.Fatalf("active faults = %d, want 1", len(paused.ActiveFaults))
	}

	_, err = engine.Step()
	if !errors.Is(err, ErrMissionPaused) {
		t.Fatalf("Step() error = %v, want ErrMissionPaused", err)
	}

	afterRejectedStep := engine.Snapshot()

	if afterRejectedStep.Rover.Position != positionBeforeFault {
		t.Errorf("rover moved while sensor fault was active")
	}

	if afterRejectedStep.Rover.BatteryPercent != batteryBeforeFault {
		t.Errorf("battery changed while movement was blocked")
	}

	if afterRejectedStep.Steps != stepsBeforeFault {
		t.Errorf("step count changed while movement was blocked")
	}

	_, err = engine.Resume()
	if !errors.Is(err, ErrActiveFaults) {
		t.Fatalf("Resume() error = %v, want ErrActiveFaults", err)
	}

	recovered, err := engine.RecoverSensorFailure(fault.ComponentNavigationSensor)
	if err != nil {
		t.Fatalf("RecoverSensorFailure() error: %v", err)
	}

	if recovered.Status != fault.StatusRecovered {
		t.Errorf("recovered fault status = %q, want %q", recovered.Status, fault.StatusRecovered)
	}

	afterRecovery := engine.Snapshot()

	// Recovery is not an automatic resume; an explicit command is required.
	if afterRecovery.Status != StatusPaused {
		t.Errorf("status after recovery = %q, want %q", afterRecovery.Status, StatusPaused)
	}

	if len(afterRecovery.ActiveFaults) != 0 {
		t.Errorf("active faults after recovery = %d, want 0", len(afterRecovery.ActiveFaults))
	}

	if len(afterRecovery.FaultHistory) != 1 {
		t.Fatalf("fault history length = %d, want 1", len(afterRecovery.FaultHistory))
	}

	if _, err := engine.Resume(); err != nil {
		t.Fatalf("Resume() after recovery error: %v", err)
	}

	resumed := engine.Snapshot()
	if resumed.Status != StatusRunning {
		t.Fatalf("status after resume = %q, want %q", resumed.Status, StatusRunning)
	}

	next, err := engine.Step()
	if err != nil {
		t.Fatalf("Step() after recovery error: %v", err)
	}

	if next.Steps != stepsBeforeFault+1 {
		t.Errorf("steps after resumed movement = %d, want %d", next.Steps, stepsBeforeFault+1)
	}

	if next.Rover.Position == positionBeforeFault {
		t.Error("rover did not move after recovery and resume")
	}
}

func TestMultipleSensorFaultsMustAllBeRecovered(t *testing.T) {
	start := world.Point{X: 0, Y: 0}

	engine, _ := newTestEngine(t, 5, 3, start, 100)

	if err := engine.Plan(world.Point{X: 4, Y: 0}); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Step(); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.InjectSensorFailure(fault.ComponentNavigationSensor); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.InjectSensorFailure(fault.ComponentBatterySensor); err != nil {
		t.Fatal(err)
	}

	if got := len(engine.Snapshot().ActiveFaults); got != 2 {
		t.Fatalf("active faults = %d, want 2", got)
	}

	if _, err := engine.RecoverSensorFailure(fault.ComponentNavigationSensor); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Resume(); !errors.Is(err, ErrActiveFaults) {
		t.Fatalf("Resume() error = %v, want ErrActiveFaults", err)
	}

	if _, err := engine.RecoverSensorFailure(fault.ComponentBatterySensor); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Resume(); err != nil {
		t.Fatalf("Resume() after all faults recovered: %v", err)
	}

	snapshot := engine.Snapshot()

	if snapshot.Status != StatusRunning {
		t.Errorf("status = %q, want %q", snapshot.Status, StatusRunning)
	}

	if len(snapshot.ActiveFaults) != 0 {
		t.Errorf("active faults = %d, want 0", len(snapshot.ActiveFaults))
	}

	if len(snapshot.FaultHistory) != 2 {
		t.Errorf("fault history = %d, want 2", len(snapshot.FaultHistory))
	}
}

func TestSensorFailurePausesPlannedMission(t *testing.T) {
	engine, _ := newTestEngine(
		t,
		4,
		3,
		world.Point{X: 0, Y: 0},
		100,
	)

	if err := engine.Plan(world.Point{X: 3, Y: 0}); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.InjectSensorFailure(fault.ComponentObstacleSensor); err != nil {
		t.Fatal(err)
	}

	snapshot := engine.Snapshot()

	if snapshot.Status != StatusPaused {
		t.Fatalf("status = %q, want %q", snapshot.Status, StatusPaused)
	}

	if snapshot.Steps != 0 {
		t.Errorf("steps = %d, want 0", snapshot.Steps)
	}

	if _, err := engine.Resume(); !errors.Is(err, ErrActiveFaults) {
		t.Fatalf("Resume() error = %v, want ErrActiveFaults", err)
	}

	if _, err := engine.RecoverSensorFailure(fault.ComponentObstacleSensor); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Resume(); err != nil {
		t.Fatalf("Resume() after recovery error: %v", err)
	}

	if _, err := engine.Step(); err != nil {
		t.Fatalf("first Step() after recovery error: %v", err)
	}

	if engine.Snapshot().Rover.Position == (world.Point{X: 0, Y: 0}) {
		t.Error("rover did not start moving after recovery")
	}
}
