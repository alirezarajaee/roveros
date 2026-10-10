package simulation

import (
	"errors"
	"fmt"
	"sync"

	"github.com/alirezarajaee/roveros/internal/navigation"
	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/world"
)

var (
	ErrNoMission        = errors.New("no mission has been planned")
	ErrMissionCompleted = errors.New("mission is already completed")
	ErrMissionFailed    = errors.New("mission has failed")
	ErrMissionPaused    = errors.New("mission is paused")
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

// Snapshot represents a copy of the simulation state.
type Snapshot struct {
	Rover     rover.Rover   `json:"rover"`
	Status    Status        `json:"status"`
	Target    world.Point   `json:"target"`
	HasTarget bool          `json:"has_target"`
	Path      []world.Point `json:"path"`
	NextStep  int           `json:"next_step"`
	Steps     int           `json:"steps"`
	LastError string        `json:"last_error,omitempty"`
}

// Engine coordinates path planning and rover movement.
type Engine struct {
	mu sync.RWMutex

	grid      *world.Grid
	rover     rover.Rover
	target    world.Point
	hasTarget bool
	path      []world.Point
	nextStep  int
	steps     int
	status    Status
	lastError string
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

	return &Engine{
		grid:   grid,
		rover:  initialRover,
		status: StatusIdle,
	}, nil
}

// Plan calculates a route to the destination.
// A failed planning attempt preserves the current mission.
func (e *Engine) Plan(destination world.Point) error {
	if e == nil {
		return fmt.Errorf("engine is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	path, err := navigation.FindPath(e.grid, e.rover.Position, destination)
	if err != nil {
		return fmt.Errorf("plan mission: %w", err)
	}

	e.target = destination
	e.hasTarget = true
	e.path = append([]world.Point(nil), path...)
	e.nextStep = 1
	e.steps = 0
	e.lastError = ""
	e.rover.Status = rover.StatusIdle

	if len(path) == 1 {
		e.status = StatusCompleted
		return nil
	}

	e.status = StatusPlanned
	return nil
}

// Step advances the rover by one cell along its planned route.
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

	if e.nextStep >= len(e.path) {
		e.status = StatusCompleted
		e.rover.Status = rover.StatusIdle
		return e.snapshotLocked(), nil
	}

	destination := e.path[e.nextStep]
	if err := e.rover.MoveTo(e.grid, destination); err != nil {
		e.status = StatusFailed
		e.lastError = err.Error()
		e.rover.Status = rover.StatusIdle

		return e.snapshotLocked(), fmt.Errorf("execute mission step: %w", err)
	}

	e.nextStep++
	e.steps++

	if e.rover.Position == e.target {
		e.status = StatusCompleted
		e.rover.Status = rover.StatusIdle
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

	return e.snapshotLocked(), nil
}

// Resume resumes a paused mission without resetting its progress.
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

	e.rover.Status = rover.StatusIdle

	if e.steps == 0 {
		e.status = StatusPlanned
	} else {
		e.status = StatusRunning
	}

	return e.snapshotLocked(), nil
}

// Snapshot returns a copy of the current simulation state.
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
		Rover:     e.rover,
		Status:    e.status,
		Target:    e.target,
		HasTarget: e.hasTarget,
		Path:      append([]world.Point(nil), e.path...),
		NextStep:  e.nextStep,
		Steps:     e.steps,
		LastError: e.lastError,
	}
}
