package fault

import (
	"errors"
	"testing"
)

func TestInjectSensorFailure(t *testing.T) {
	manager := NewManager()

	event, err := manager.InjectSensorFailure(ComponentNavigationSensor)
	if err != nil {
		t.Fatalf("InjectSensorFailure() error: %v", err)
	}

	if event.ID == 0 {
		t.Error("fault ID must be non-zero")
	}

	if event.Type != TypeSensorFailure {
		t.Errorf("Type = %q, want %q", event.Type, TypeSensorFailure)
	}

	if event.Component != ComponentNavigationSensor {
		t.Errorf(
			"Component = %q, want %q",
			event.Component,
			ComponentNavigationSensor,
		)
	}

	if event.Status != StatusActive {
		t.Errorf("Status = %q, want %q", event.Status, StatusActive)
	}

	active := manager.ActiveFaults()
	if len(active) != 1 {
		t.Fatalf("ActiveFaults() length = %d, want 1", len(active))
	}

	history := manager.History()
	if len(history) != 1 {
		t.Fatalf("History() length = %d, want 1", len(history))
	}
}

func TestInjectRejectsInvalidComponent(t *testing.T) {
	manager := NewManager()

	_, err := manager.InjectSensorFailure(Component("unknown_sensor"))
	if !errors.Is(err, ErrInvalidComponent) {
		t.Fatalf("error = %v, want ErrInvalidComponent", err)
	}

	if len(manager.History()) != 0 {
		t.Error("invalid component must not be recorded")
	}
}

func TestInjectRejectsDuplicateActiveFailure(t *testing.T) {
	manager := NewManager()

	if _, err := manager.InjectSensorFailure(ComponentBatterySensor); err != nil {
		t.Fatalf("first injection error: %v", err)
	}

	_, err := manager.InjectSensorFailure(ComponentBatterySensor)
	if !errors.Is(err, ErrFaultAlreadyActive) {
		t.Fatalf("error = %v, want ErrFaultAlreadyActive", err)
	}

	if got := len(manager.History()); got != 1 {
		t.Errorf("History() length = %d, want 1", got)
	}
}

func TestRecoverSensorFailure(t *testing.T) {
	manager := NewManager()

	if _, err := manager.InjectSensorFailure(ComponentObstacleSensor); err != nil {
		t.Fatal(err)
	}

	recovered, err := manager.RecoverSensorFailure(ComponentObstacleSensor)
	if err != nil {
		t.Fatalf("RecoverSensorFailure() error: %v", err)
	}

	if recovered.Status != StatusRecovered {
		t.Errorf(
			"Status = %q, want %q",
			recovered.Status,
			StatusRecovered,
		)
	}

	if got := len(manager.ActiveFaults()); got != 0 {
		t.Errorf("ActiveFaults() length = %d, want 0", got)
	}

	history := manager.History()
	if len(history) != 1 {
		t.Fatalf("History() length = %d, want 1", len(history))
	}

	if history[0].Status != StatusRecovered {
		t.Errorf(
			"history status = %q, want %q",
			history[0].Status,
			StatusRecovered,
		)
	}
}

func TestRecoverRejectsMissingActiveFailure(t *testing.T) {
	manager := NewManager()

	_, err := manager.RecoverSensorFailure(ComponentNavigationSensor)
	if !errors.Is(err, ErrNoActiveFault) {
		t.Fatalf("error = %v, want ErrNoActiveFault", err)
	}
}

func TestSensorCanFailAgainAfterRecovery(t *testing.T) {
	manager := NewManager()

	first, err := manager.InjectSensorFailure(ComponentNavigationSensor)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.RecoverSensorFailure(ComponentNavigationSensor); err != nil {
		t.Fatal(err)
	}

	second, err := manager.InjectSensorFailure(ComponentNavigationSensor)
	if err != nil {
		t.Fatalf("second injection error: %v", err)
	}

	if second.ID <= first.ID {
		t.Errorf("second ID = %d, first ID = %d; want increasing IDs", second.ID, first.ID)
	}

	if got := len(manager.History()); got != 2 {
		t.Errorf("History() length = %d, want 2", got)
	}

	if got := len(manager.ActiveFaults()); got != 1 {
		t.Errorf("ActiveFaults() length = %d, want 1", got)
	}
}

func TestHistoryReturnsIndependentCopy(t *testing.T) {
	manager := NewManager()

	if _, err := manager.InjectSensorFailure(ComponentBatterySensor); err != nil {
		t.Fatal(err)
	}

	history := manager.History()
	history[0].Status = StatusRecovered
	history[0].Message = "modified externally"

	actual := manager.History()
	if actual[0].Status != StatusActive {
		t.Errorf("internal status changed to %q", actual[0].Status)
	}

	if actual[0].Message == "modified externally" {
		t.Error("external modification changed internal history")
	}
}

func TestNilManagerReturnsErrorsOrEmptyResults(t *testing.T) {
	var manager *Manager

	if _, err := manager.InjectSensorFailure(ComponentBatterySensor); !errors.Is(err, ErrNilManager) {
		t.Errorf("InjectSensorFailure() error = %v, want ErrNilManager", err)
	}

	if _, err := manager.RecoverSensorFailure(ComponentBatterySensor); !errors.Is(err, ErrNilManager) {
		t.Errorf("RecoverSensorFailure() error = %v, want ErrNilManager", err)
	}

	if got := manager.ActiveFaults(); got != nil {
		t.Errorf("ActiveFaults() = %v, want nil", got)
	}

	if got := manager.History(); got != nil {
		t.Errorf("History() = %v, want nil", got)
	}
}
