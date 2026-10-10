package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/alirezarajaee/roveros/internal/fault"
	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/simulation"
	"github.com/alirezarajaee/roveros/internal/world"
)

// Server exposes the rover simulation through an HTTP API.
type Server struct {
	mu     sync.RWMutex
	engine *simulation.Engine
}

// NewServer creates an API server with a default mission.
func NewServer() (*Server, error) {
	engine, err := newDefaultEngine()
	if err != nil {
		return nil, err
	}

	return &Server{engine: engine}, nil
}

func newDefaultEngine() (*simulation.Engine, error) {
	grid, err := world.NewGrid(20, 10)
	if err != nil {
		return nil, err
	}

	// Create a wall with an opening near the bottom.
	for y := 0; y < 8; y++ {
		if err := grid.SetBlocked(world.Point{X: 8, Y: y}, true); err != nil {
			return nil, err
		}
	}

	// Add rocks to the terrain.
	obstacles := []world.Point{
		{X: 4, Y: 2},
		{X: 4, Y: 3},
		{X: 4, Y: 4},
		{X: 12, Y: 4},
		{X: 12, Y: 5},
	}

	for _, point := range obstacles {
		if err := grid.SetBlocked(point, true); err != nil {
			return nil, err
		}
	}

	initialRover, err := rover.New(world.Point{X: 1, Y: 1}, 100)
	if err != nil {
		return nil, err
	}

	engine, err := simulation.New(grid, initialRover)
	if err != nil {
		return nil, err
	}

	if err := engine.Plan(world.Point{X: 15, Y: 7}); err != nil {
		return nil, fmt.Errorf("plan initial mission: %w", err)
	}

	return engine, nil
}

// ServeHTTP routes requests to the appropriate API handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	if origin != "" && !allowedOrigin(origin) {
		if r.Method == http.MethodOptions {
			writeError(w, http.StatusForbidden, "origin not allowed")
			return
		}
	} else if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, OPTIONS",
		)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.URL.Path {
	case "/api/health":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "roveros-api",
		})

	case "/api/state":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		respondJSON(w, http.StatusOK, s.engine.Snapshot())

	case "/api/mission/start":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.startMission(w)

	case "/api/mission/step":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.stepMission(w)

	case "/api/mission/pause":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.pauseMission(w)

	case "/api/mission/resume":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.resumeMission(w)

	case "/api/mission/reset":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.resetMission(w)

	case "/api/faults/inject":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.injectFault(w, r)

	case "/api/faults/recover":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}

		s.recoverFault(w, r)

	default:
		writeError(w, http.StatusNotFound, "endpoint not found")
	}
}

func (s *Server) startMission(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current := s.engine.Snapshot()

	switch current.Status {
	case simulation.StatusPlanned:
		snapshot, err := s.engine.Step()
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, snapshot)

	case simulation.StatusRunning:
		// Repeated start requests do not advance the mission.
		respondJSON(w, http.StatusOK, current)

	default:
		writeError(
			w,
			http.StatusConflict,
			fmt.Sprintf(
				"cannot start mission with status %q",
				current.Status,
			),
		)
	}
}

func (s *Server) stepMission(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, err := s.engine.Step()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, snapshot)
}

func (s *Server) pauseMission(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, err := s.engine.Pause()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, snapshot)
}

func (s *Server) resumeMission(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, err := s.engine.Resume()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, snapshot)
}

func (s *Server) resetMission(w http.ResponseWriter) {
	// Build the replacement before changing the current mission.
	engine, err := newDefaultEngine()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.Lock()
	s.engine = engine
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, engine.Snapshot())
}

type sensorFaultRequest struct {
	Component fault.Component `json:"component"`
}

func decodeSensorFaultRequest(
	w http.ResponseWriter,
	r *http.Request,
) (fault.Component, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request sensorFaultRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return "", false
	}

	// Reject trailing JSON values or other unexpected content.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(
			w,
			http.StatusBadRequest,
			"request must contain one JSON object",
		)
		return "", false
	}

	return request.Component, true
}

func (s *Server) injectFault(w http.ResponseWriter, r *http.Request) {
	component, ok := decodeSensorFaultRequest(w, r)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := s.engine.InjectSensorFailure(component)
	if err != nil {
		status := http.StatusBadRequest

		if errors.Is(err, fault.ErrFaultAlreadyActive) {
			status = http.StatusConflict
		}

		writeError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, s.engine.Snapshot())
}

func (s *Server) recoverFault(w http.ResponseWriter, r *http.Request) {
	component, ok := decodeSensorFaultRequest(w, r)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := s.engine.RecoverSensorFailure(component)
	if err != nil {
		status := http.StatusBadRequest

		if errors.Is(err, fault.ErrNoActiveFault) {
			status = http.StatusConflict
		}

		writeError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, s.engine.Snapshot())
}

func allowedOrigin(origin string) bool {
	return origin == "http://localhost:5173" ||
		origin == "http://127.0.0.1:5173"
}

func requireMethod(
	w http.ResponseWriter,
	r *http.Request,
	method string,
) bool {
	if r.Method == method {
		return true
	}

	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		// The response has already started; no alternative status
		// code can safely be sent at this point.
		return
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
