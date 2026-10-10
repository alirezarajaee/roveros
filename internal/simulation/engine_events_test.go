package simulation

import (
	"testing"
	"time"

	"github.com/alirezarajaee/roveros/internal/fault"
	"github.com/alirezarajaee/roveros/internal/mission"
	"github.com/alirezarajaee/roveros/internal/world"
)

func hasEventType(events []mission.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}

	return false
}

func TestEngineRecordsMissionLifecycleEvents(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 2, Y: 0}

	engine, _ := newTestEngine(t, 5, 3, start, 100)

	initial := engine.Snapshot()
	if !hasEventType(initial.Events, "mission_initialized") {
		t.Fatal("initialization event was not recorded")
	}

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	if _, err := engine.Step(); err != nil {
		t.Fatalf("first Step() error: %v", err)
	}

	if _, err := engine.Pause(); err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	if _, err := engine.Resume(); err != nil {
		t.Fatalf("Resume() error: %v", err)
	}

	if _, err := engine.Step(); err != nil {
		t.Fatalf("final Step() error: %v", err)
	}

	snapshot := engine.Snapshot()

	if snapshot.Status != StatusCompleted {
		t.Fatalf(
			"Status = %q, want %q",
			snapshot.Status,
			StatusCompleted,
		)
	}

	requiredTypes := []string{
		"mission_initialized",
		"mission_planned",
		"mission_started",
		"rover_moved",
		"mission_paused",
		"mission_resumed",
		"mission_completed",
	}

	for _, eventType := range requiredTypes {
		if !hasEventType(snapshot.Events, eventType) {
			t.Errorf("event type %q was not recorded", eventType)
		}
	}

	if len(snapshot.Events) < len(requiredTypes) {
		t.Fatalf(
			"event count = %d, want at least %d",
			len(snapshot.Events),
			len(requiredTypes),
		)
	}

	for i, event := range snapshot.Events {
		if event.ID == 0 {
			t.Errorf("event %d has zero ID", i)
		}

		if event.Type == "" {
			t.Errorf("event %d has empty type", i)
		}

		if event.Message == "" {
			t.Errorf("event %d has empty message", i)
		}

		if event.Timestamp.IsZero() {
			t.Errorf("event %d has zero timestamp", i)
		}

		if event.Timestamp.Location() != time.UTC {
			t.Errorf("event %d timestamp is not UTC", i)
		}

		if i > 0 && event.ID <= snapshot.Events[i-1].ID {
			t.Errorf(
				"event IDs are not increasing: %d followed by %d",
				snapshot.Events[i-1].ID,
				event.ID,
			)
		}
	}
}

func TestEngineRecordsSensorFaultAndRecoveryEvents(t *testing.T) {
	start := world.Point{X: 0, Y: 0}
	target := world.Point{X: 3, Y: 0}

	engine, _ := newTestEngine(t, 5, 3, start, 100)

	if err := engine.Plan(target); err != nil {
		t.Fatalf("Plan() error: %v", err)
	}

	if _, err := engine.Step(); err != nil {
		t.Fatalf("first Step() error: %v", err)
	}

	if _, err := engine.InjectSensorFailure(
		fault.ComponentNavigationSensor,
	); err != nil {
		t.Fatalf("InjectSensorFailure() error: %v", err)
	}

	afterFault := engine.Snapshot()

	if afterFault.Status != StatusPaused {
		t.Fatalf(
			"status after fault = %q, want %q",
			afterFault.Status,
			StatusPaused,
		)
	}

	if !hasEventType(afterFault.Events, "sensor_fault_injected") {
		t.Fatal("sensor fault injection event was not recorded")
	}

	if !hasEventType(afterFault.Events, "mission_paused") {
		t.Fatal("safety pause event was not recorded")
	}

	if _, err := engine.RecoverSensorFailure(
		fault.ComponentNavigationSensor,
	); err != nil {
		t.Fatalf("RecoverSensorFailure() error: %v", err)
	}

	afterRecovery := engine.Snapshot()

	if !hasEventType(afterRecovery.Events, "sensor_fault_recovered") {
		t.Fatal("sensor recovery event was not recorded")
	}

	if afterRecovery.Status != StatusPaused {
		t.Fatalf(
			"status after recovery = %q, want %q",
			afterRecovery.Status,
			StatusPaused,
		)
	}

	if _, err := engine.Resume(); err != nil {
		t.Fatalf("Resume() after recovery error: %v", err)
	}

	final := engine.Snapshot()

	if !hasEventType(final.Events, "mission_resumed") {
		t.Fatal("mission resume event was not recorded")
	}

	if final.Status != StatusRunning {
		t.Errorf(
			"status after resume = %q, want %q",
			final.Status,
			StatusRunning,
		)
	}
}

func TestSnapshotReturnsIndependentEventHistory(t *testing.T) {
	engine, _ := newTestEngine(
		t,
		4,
		3,
		world.Point{X: 0, Y: 0},
		100,
	)

	first := engine.Snapshot()

	if len(first.Events) == 0 {
		t.Fatal("expected initialization event")
	}

	originalMessage := first.Events[0].Message

	// Mutating a snapshot must not modify the engine's internal history.
	first.Events[0].Message = "modified outside the engine"

	second := engine.Snapshot()

	if second.Events[0].Message != originalMessage {
		t.Errorf(
			"internal event message changed to %q",
			second.Events[0].Message,
		)
	}
}
