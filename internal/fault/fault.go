package fault

import (
	"errors"
	"fmt"
	"sync"
)

type Type string

const (
	TypeSensorFailure Type = "sensor_failure"
)

type Component string

const (
	ComponentNavigationSensor Component = "navigation_sensor"
	ComponentObstacleSensor   Component = "obstacle_sensor"
	ComponentBatterySensor    Component = "battery_sensor"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusRecovered Status = "recovered"
)

var (
	ErrNilManager         = errors.New("fault manager is nil")
	ErrInvalidComponent   = errors.New("invalid sensor component")
	ErrFaultAlreadyActive = errors.New("sensor fault is already active")
	ErrNoActiveFault      = errors.New("no active fault exists for this sensor")
)

// Fault records a simulated sensor failure and its recovery status.
type Fault struct {
	ID        uint64    `json:"id"`
	Type      Type      `json:"type"`
	Component Component `json:"component"`
	Status    Status    `json:"status"`
	Message   string    `json:"message"`
}

// Manager maintains the fault history for a simulation.
type Manager struct {
	mu      sync.RWMutex
	nextID  uint64
	history []Fault
}

// NewManager creates an empty fault manager.
func NewManager() *Manager {
	return &Manager{}
}

// InjectSensorFailure records a sensor failure.
// An active failure for the same sensor cannot be duplicated.
func (m *Manager) InjectSensorFailure(component Component) (Fault, error) {
	if m == nil {
		return Fault{}, ErrNilManager
	}

	if !validSensorComponent(component) {
		return Fault{}, fmt.Errorf("%w: %q", ErrInvalidComponent, component)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.history {
		if existing.Type == TypeSensorFailure &&
			existing.Component == component &&
			existing.Status == StatusActive {
			return Fault{}, fmt.Errorf(
				"%w: %s",
				ErrFaultAlreadyActive,
				component,
			)
		}
	}

	m.nextID++

	event := Fault{
		ID:        m.nextID,
		Type:      TypeSensorFailure,
		Component: component,
		Status:    StatusActive,
		Message:   fmt.Sprintf("%s failure detected", component),
	}

	m.history = append(m.history, event)

	return event, nil
}

// RecoverSensorFailure marks an active failure as recovered.
func (m *Manager) RecoverSensorFailure(component Component) (Fault, error) {
	if m == nil {
		return Fault{}, ErrNilManager
	}

	if !validSensorComponent(component) {
		return Fault{}, fmt.Errorf("%w: %q", ErrInvalidComponent, component)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := len(m.history) - 1; i >= 0; i-- {
		event := &m.history[i]

		if event.Type == TypeSensorFailure &&
			event.Component == component &&
			event.Status == StatusActive {
			event.Status = StatusRecovered
			event.Message = fmt.Sprintf("%s failure recovered", component)

			return *event, nil
		}
	}

	return Fault{}, fmt.Errorf("%w: %s", ErrNoActiveFault, component)
}

// ActiveFaults returns a copy of all currently active faults.
func (m *Manager) ActiveFaults() []Fault {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]Fault, 0)

	for _, event := range m.history {
		if event.Status == StatusActive {
			active = append(active, event)
		}
	}

	return active
}

// History returns a copy of the complete fault history.
func (m *Manager) History() []Fault {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]Fault(nil), m.history...)
}

func validSensorComponent(component Component) bool {
	switch component {
	case ComponentNavigationSensor,
		ComponentObstacleSensor,
		ComponentBatterySensor:
		return true
	default:
		return false
	}
}
