package simulation

import (
	"errors"
	"fmt"
	"sync"

	"github.com/alirezarajaee/roveros/internal/fault"
	"github.com/alirezarajaee/roveros/internal/mission"
	"github.com/alirezarajaee/roveros/internal/navigation"
	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/world"
)

var (
	ErrNoMission        = errors.New("no mission has been planned")
	ErrMissionCompleted = errors.New("mission is already completed")
	ErrMissionFailed    = errors.New("mission has failed")
	ErrMissionPaused    = errors.New("mission is paused")
	ErrActiveFaults     = errors.New("active sensor faults prevent mission execution")
)

type Status string

const (
	StatusIdle      Status = "idle"
	StatusPlanned   Status = "planned"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Snapshot represents a copy of the current simulation state.
type Snapshot struct {
	Rover        rover.Rover     `json:"rover"`
	Status       Status          `json:"status"`
	Target       world.Point     `json:"target"`
	HasTarget    bool            `json:"has_target"`
	Path         []world.Point   `json:"path"`
	NextStep     int             `json:"next_step"`
	Steps        int             `json:"steps"`
	LastError    string          `json:"last_error,omitempty"`
	SafetyReason string          `json:"safety_reason,omitempty"`
	ActiveFaults []fault.Fault   `json:"active_faults"`
	FaultHistory []fault.Fault   `json:"fault_history"`
	Events       []mission.Event `json:"events"`
}

// Engine coordinates mission planning, movement, fault handling,
// and mission event recording.
type Engine struct {
	mu sync.RWMutex

	grid         *world.Grid
	rover        rover.Rover
	target       world.Point
	hasTarget    bool
	path         []world.Point
	nextStep     int
	steps        int
	status       Status
	lastError    string
	safetyReason string

	faultManager *fault.Manager
	eventLog     *mission.Log
}

// New creates a simulation engine with an idle rover on a walkable cell.
func New(grid *world.Grid, initialRover rover.Rover) (*Engine, error) {
	if grid == nil {
		return nil, fmt.Errorf("grid is nil")
	}

	if err := initialRover.Validate(); err != nil {
		return nil, fmt.Errorf("invalid initial rover: %w", err)
	}

	if initialRover.Status != rover.StatusIdle {
		return nil, fmt.Errorf("initial rover must be idle")
	}

	if !grid.IsWalkable(initialRover.Position) {
		return nil, fmt.Errorf("initial rover position is not walkable")
	}

	engine := &Engine{
		grid:         grid,
		rover:        initialRover,
		status:       StatusIdle,
		faultManager: fault.NewManager(),
		eventLog:     mission.NewLog(mission.DefaultEventCapacity),
	}

	engine.recordEvent(
		"mission_initialized",
		"info",
		"Simulation engine initialized",
	)

	return engine, nil
}

// Plan calculates a route to the destination.
// A failed planning attempt preserves the existing mission state.
func (e *Engine) Plan(destination world.Point) error {
	if e == nil {
		return fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.faultManager.ActiveFaults()) > 0 {
		e.recordEvent(
			"mission_plan_blocked",
			"warning",
			"Mission planning rejected because sensor faults are active",
		)
		return ErrActiveFaults
	}

	path, err := navigation.FindPath(e.grid, e.rover.Position, destination)
	if err != nil {
		e.recordEvent(
			"mission_plan_failed",
			"error",
			fmt.Sprintf("Unable to plan route: %v", err),
		)
		return fmt.Errorf("plan mission: %w", err)
	}

	e.target = destination
	e.hasTarget = true
	e.path = append([]world.Point(nil), path...)
	e.nextStep = 1
	e.steps = 0
	e.lastError = ""
	e.safetyReason = ""
	e.rover.Status = rover.StatusIdle

	if len(path) == 1 {
		e.status = StatusCompleted
	} else {
		e.status = StatusPlanned
	}

	e.recordEvent(
		"mission_planned",
		"info",
		fmt.Sprintf(
			"Mission planned to (%d, %d) with %d moves",
			destination.X,
			destination.Y,
			len(path)-1,
		),
	)

	if e.status == StatusCompleted {
		e.recordEvent(
			"mission_completed",
			"info",
			"Mission completed because the rover was already at the target",
		)
	}

	return nil
}

// Step advances the rover by one cell along the planned route.
func (e *Engine) Step() (Snapshot, error) {
	if e == nil {
		return Snapshot{}, fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.hasTarget || e.status == StatusIdle {
		return e.snapshotLocked(), ErrNoMission
	}

	if e.status == StatusPaused {
		return e.snapshotLocked(), ErrMissionPaused
	}

	if e.status == StatusCompleted {
		return e.snapshotLocked(), ErrMissionCompleted
	}

	if e.status == StatusFailed {
		return e.snapshotLocked(), fmt.Errorf(
			"%w: %s",
			ErrMissionFailed,
			e.lastError,
		)
	}

	activeFaults := e.faultManager.ActiveFaults()
	if len(activeFaults) > 0 {
		e.status = StatusPaused
		e.rover.Status = rover.StatusPaused
		e.safetyReason = fmt.Sprintf(
			"Mission held for safety: %d active sensor fault(s)",
			len(activeFaults),
		)

		e.recordEvent(
			"mission_paused",
			"warning",
			e.safetyReason,
		)

		return e.snapshotLocked(), ErrMissionPaused
	}

	if e.nextStep >= len(e.path) {
		e.status = StatusCompleted
		e.rover.Status = rover.StatusIdle
		e.recordEvent(
			"mission_completed",
			"info",
			"Rover reached the mission target",
		)
		return e.snapshotLocked(), nil
	}

	wasPlanned := e.status == StatusPlanned
	destination := e.path[e.nextStep]

	if err := e.rover.MoveTo(e.grid, destination); err != nil {
		e.status = StatusFailed
		e.lastError = err.Error()
		e.rover.Status = rover.StatusIdle

		e.recordEvent(
			"mission_failed",
			"error",
			fmt.Sprintf("Rover movement failed: %v", err),
		)

		return e.snapshotLocked(), fmt.Errorf("execute mission step: %w", err)
	}

	e.nextStep++
	e.steps++

	if wasPlanned {
		e.recordEvent(
			"mission_started",
			"info",
			"Mission execution started",
		)
	}

	e.recordEvent(
		"rover_moved",
		"info",
		fmt.Sprintf(
			"Rover moved to (%d, %d); battery %.1f%%",
			e.rover.Position.X,
			e.rover.Position.Y,
			e.rover.BatteryPercent,
		),
	)

	if e.rover.Position == e.target {
		e.status = StatusCompleted
		e.rover.Status = rover.StatusIdle

		e.recordEvent(
			"mission_completed",
			"info",
			fmt.Sprintf(
				"Rover reached target (%d, %d) after %d moves",
				e.target.X,
				e.target.Y,
				e.steps,
			),
		)
	} else {
		e.status = StatusRunning
	}

	return e.snapshotLocked(), nil
}

// Pause suspends a planned or running mission.
func (e *Engine) Pause() (Snapshot, error) {
	if e == nil {
		return Snapshot{}, fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != StatusPlanned && e.status != StatusRunning {
		return e.snapshotLocked(), fmt.Errorf(
			"cannot pause mission with status %q",
			e.status,
		)
	}

	e.status = StatusPaused
	e.rover.Status = rover.StatusPaused

	e.recordEvent(
		"mission_paused",
		"info",
		fmt.Sprintf("Mission paused at (%d, %d)", e.rover.Position.X, e.rover.Position.Y),
	)

	return e.snapshotLocked(), nil
}

// Resume resumes a paused mission only when every sensor fault is recovered.
func (e *Engine) Resume() (Snapshot, error) {
	if e == nil {
		return Snapshot{}, fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != StatusPaused {
		return e.snapshotLocked(), fmt.Errorf(
			"cannot resume mission with status %q",
			e.status,
		)
	}

	activeFaults := e.faultManager.ActiveFaults()
	if len(activeFaults) > 0 {
		e.safetyReason = fmt.Sprintf(
			"Mission held for safety: %d active sensor fault(s)",
			len(activeFaults),
		)

		e.recordEvent(
			"mission_resume_blocked",
			"warning",
			e.safetyReason,
		)

		return e.snapshotLocked(), fmt.Errorf(
			"%w: %d active sensor fault(s)",
			ErrActiveFaults,
			len(activeFaults),
		)
	}

	e.safetyReason = ""
	e.rover.Status = rover.StatusIdle

	if e.steps == 0 {
		e.status = StatusPlanned
	} else {
		e.status = StatusRunning
	}

	e.recordEvent(
		"mission_resumed",
		"info",
		fmt.Sprintf(
			"Mission resumed from (%d, %d)",
			e.rover.Position.X,
			e.rover.Position.Y,
		),
	)

	return e.snapshotLocked(), nil
}

// InjectSensorFailure records a fault and pauses an active mission for safety.
func (e *Engine) InjectSensorFailure(
	component fault.Component,
) (fault.Fault, error) {
	if e == nil {
		return fault.Fault{}, fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	event, err := e.faultManager.InjectSensorFailure(component)
	if err != nil {
		e.recordEvent(
			"sensor_fault_injection_rejected",
			"warning",
			fmt.Sprintf("Sensor fault injection rejected: %v", err),
		)
		return fault.Fault{}, err
	}

	e.recordEvent(
		"sensor_fault_injected",
		"warning",
		fmt.Sprintf("%s failure injected", component),
	)

	activeFaults := e.faultManager.ActiveFaults()
	if len(activeFaults) > 0 {
		e.safetyReason = fmt.Sprintf(
			"Mission held for safety: %d active sensor fault(s)",
			len(activeFaults),
		)

		if e.status == StatusPlanned || e.status == StatusRunning {
			e.status = StatusPaused
			e.rover.Status = rover.StatusPaused

			e.recordEvent(
				"mission_paused",
				"warning",
				e.safetyReason,
			)
		}
	}

	return event, nil
}

// RecoverSensorFailure records recovery.
// A paused mission must be explicitly resumed after all faults are cleared.
func (e *Engine) RecoverSensorFailure(
	component fault.Component,
) (fault.Fault, error) {
	if e == nil {
		return fault.Fault{}, fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	event, err := e.faultManager.RecoverSensorFailure(component)
	if err != nil {
		e.recordEvent(
			"sensor_fault_recovery_rejected",
			"warning",
			fmt.Sprintf("Sensor fault recovery rejected: %v", err),
		)
		return fault.Fault{}, err
	}

	e.recordEvent(
		"sensor_fault_recovered",
		"info",
		fmt.Sprintf("%s failure recovered", component),
	)

	activeFaults := e.faultManager.ActiveFaults()
	if len(activeFaults) == 0 {
		e.safetyReason = ""
	} else {
		e.safetyReason = fmt.Sprintf(
			"Mission held for safety: %d active sensor fault(s)",
			len(activeFaults),
		)
	}

	return event, nil
}

// Snapshot returns a copy of the current state and retained histories.
func (e *Engine) Snapshot() Snapshot {
	if e == nil {
		return Snapshot{}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.snapshotLocked()
}

func (e *Engine) snapshotLocked() Snapshot {
	return Snapshot{
		Rover:        e.rover,
		Status:       e.status,
		Target:       e.target,
		HasTarget:    e.hasTarget,
		Path:         append([]world.Point(nil), e.path...),
		NextStep:     e.nextStep,
		Steps:        e.steps,
		LastError:    e.lastError,
		SafetyReason: e.safetyReason,
		ActiveFaults: e.faultManager.ActiveFaults(),
		FaultHistory: e.faultManager.History(),
		Events:       e.eventLog.List(),
	}
}

func (e *Engine) recordEvent(eventType, level, message string) {
	e.eventLog.Append(eventType, level, message)
}
