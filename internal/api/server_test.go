package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alirezarajaee/roveros/internal/simulation"
	"github.com/alirezarajaee/roveros/internal/world"
)

func newTestAPIServer(t *testing.T) *Server {
	t.Helper()

	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	return server
}

func performRequest(
	t *testing.T,
	server http.Handler,
	method string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	return recorder
}

func decodeSnapshot(t *testing.T, recorder *httptest.ResponseRecorder) simulation.Snapshot {
	t.Helper()

	var snapshot simulation.Snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("failed to decode snapshot: %v; body: %s", err, recorder.Body.String())
	}

	return snapshot
}

func TestHealthEndpoint(t *testing.T) {
	server := newTestAPIServer(t)

	response := performRequest(
		t,
		server,
		http.MethodGet,
		"/api/health",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}

	if body["service"] != "roveros-api" {
		t.Errorf("service = %q, want %q", body["service"], "roveros-api")
	}
}

func TestStateEndpointReturnsInitialMission(t *testing.T) {
	server := newTestAPIServer(t)

	response := performRequest(
		t,
		server,
		http.MethodGet,
		"/api/state",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	snapshot := decodeSnapshot(t, response)

	if snapshot.Status != simulation.StatusPlanned {
		t.Errorf("status = %q, want %q", snapshot.Status, simulation.StatusPlanned)
	}

	if snapshot.Rover.Position != (world.Point{X: 1, Y: 1}) {
		t.Errorf("initial position = %+v, want (1, 1)", snapshot.Rover.Position)
	}

	if snapshot.Target != (world.Point{X: 15, Y: 7}) {
		t.Errorf("target = %+v, want (15, 7)", snapshot.Target)
	}

	if !snapshot.HasTarget {
		t.Error("HasTarget = false, want true")
	}

	if len(snapshot.Path) != 23 {
		t.Errorf("path points = %d, want 23", len(snapshot.Path))
	}

	if snapshot.Rover.BatteryPercent != 100 {
		t.Errorf("battery = %v, want 100", snapshot.Rover.BatteryPercent)
	}
}

func TestMissionLifecycleEndpoints(t *testing.T) {
	server := newTestAPIServer(t)

	// Start the mission and execute the first movement.
	response := performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/start",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", response.Code, response.Body.String())
	}

	snapshot := decodeSnapshot(t, response)

	if snapshot.Status != simulation.StatusRunning {
		t.Fatalf("status after start = %q, want %q", snapshot.Status, simulation.StatusRunning)
	}

	if snapshot.Rover.Position != (world.Point{X: 2, Y: 1}) {
		t.Errorf("position after start = %+v, want (2, 1)", snapshot.Rover.Position)
	}

	if snapshot.Steps != 1 {
		t.Errorf("steps after start = %d, want 1", snapshot.Steps)
	}

	if snapshot.Rover.BatteryPercent != 99 {
		t.Errorf("battery after start = %v, want 99", snapshot.Rover.BatteryPercent)
	}

	// Pause the mission.
	response = performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/pause",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("pause status = %d, body = %s", response.Code, response.Body.String())
	}

	snapshot = decodeSnapshot(t, response)

	if snapshot.Status != simulation.StatusPaused {
		t.Fatalf("status after pause = %q, want %q", snapshot.Status, simulation.StatusPaused)
	}

	pausedPosition := snapshot.Rover.Position
	pausedBattery := snapshot.Rover.BatteryPercent
	pausedSteps := snapshot.Steps

	// A movement command must not move a paused rover.
	response = performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/step",
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("step while paused status = %d, want %d", response.Code, http.StatusConflict)
	}

	response = performRequest(
		t,
		server,
		http.MethodGet,
		"/api/state",
	)

	snapshot = decodeSnapshot(t, response)

	if snapshot.Rover.Position != pausedPosition {
		t.Errorf("rover moved while paused: %+v", snapshot.Rover.Position)
	}

	if snapshot.Rover.BatteryPercent != pausedBattery {
		t.Errorf("battery changed while paused: %v", snapshot.Rover.BatteryPercent)
	}

	if snapshot.Steps != pausedSteps {
		t.Errorf("steps changed while paused: %d", snapshot.Steps)
	}

	// Resume and execute the next movement.
	response = performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/resume",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("resume status = %d, body = %s", response.Code, response.Body.String())
	}

	snapshot = decodeSnapshot(t, response)

	if snapshot.Status != simulation.StatusRunning {
		t.Fatalf("status after resume = %q, want %q", snapshot.Status, simulation.StatusRunning)
	}

	response = performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/step",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("step status = %d, body = %s", response.Code, response.Body.String())
	}

	snapshot = decodeSnapshot(t, response)

	if snapshot.Rover.Position != (world.Point{X: 3, Y: 1}) {
		t.Errorf("position after resume and step = %+v, want (3, 1)", snapshot.Rover.Position)
	}

	if snapshot.Steps != 2 {
		t.Errorf("steps after resume and step = %d, want 2", snapshot.Steps)
	}

	// Reset must create a fresh mission.
	response = performRequest(
		t,
		server,
		http.MethodPost,
		"/api/mission/reset",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("reset status = %d, body = %s", response.Code, response.Body.String())
	}

	snapshot = decodeSnapshot(t, response)

	if snapshot.Status != simulation.StatusPlanned {
		t.Errorf("status after reset = %q, want %q", snapshot.Status, simulation.StatusPlanned)
	}

	if snapshot.Rover.Position != (world.Point{X: 1, Y: 1}) {
		t.Errorf("position after reset = %+v, want (1, 1)", snapshot.Rover.Position)
	}

	if snapshot.Steps != 0 {
		t.Errorf("steps after reset = %d, want 0", snapshot.Steps)
	}

	if snapshot.Rover.BatteryPercent != 100 {
		t.Errorf("battery after reset = %v, want 100", snapshot.Rover.BatteryPercent)
	}
}

func TestAPIRejectsUnsupportedMethodAndUnknownRoute(t *testing.T) {
	server := newTestAPIServer(t)

	wrongMethod := performRequest(
		t,
		server,
		http.MethodPost,
		"/api/state",
	)

	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"wrong method status = %d, want %d",
			wrongMethod.Code,
			http.StatusMethodNotAllowed,
		)
	}

	unknownRoute := performRequest(
		t,
		server,
		http.MethodGet,
		"/api/does-not-exist",
	)

	if unknownRoute.Code != http.StatusNotFound {
		t.Errorf(
			"unknown route status = %d, want %d",
			unknownRoute.Code,
			http.StatusNotFound,
		)
	}
}

func TestAPICORSAllowsLocalFrontend(t *testing.T) {
	server := newTestAPIServer(t)

	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("allow-origin = %q, want localhost frontend origin", got)
	}
}

func TestAPICORSRejectsDisallowedPreflight(t *testing.T) {
	server := newTestAPIServer(t)

	request := httptest.NewRequest(http.MethodOptions, "/api/state", nil)
	request.Header.Set("Origin", "https://example.com")

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}

	if strings.TrimSpace(response.Body.String()) == "" {
		t.Error("expected JSON error response")
	}
}
