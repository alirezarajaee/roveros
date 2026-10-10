package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alirezarajaee/roveros/internal/fault"
	"github.com/alirezarajaee/roveros/internal/simulation"
)

func performFaultRequest(
	t *testing.T,
	server http.Handler,
	endpoint string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost,
		endpoint,
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	return response
}

func decodeFaultSnapshot(
	t *testing.T,
	response *httptest.ResponseRecorder,
) simulation.Snapshot {
	t.Helper()

	var snapshot simulation.Snapshot

	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf(
			"failed to decode snapshot: %v; response: %s",
			err,
			response.Body.String(),
		)
	}

	return snapshot
}

func TestFaultAPIInjectRecoverAndResume(t *testing.T) {
	server := newTestAPIServer(t)

	// Inject a navigation sensor failure.
	response := performFaultRequest(
		t,
		server,
		"/api/faults/inject",
		`{"component":"navigation_sensor"}`,
	)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"inject status = %d, want %d; body: %s",
			response.Code,
			http.StatusOK,
			response.Body.String(),
		)
	}

	snapshot := decodeFaultSnapshot(t, response)

	if snapshot.Status != simulation.StatusPaused {
		t.Fatalf(
			"mission status = %q, want %q",
			snapshot.Status,
			simulation.StatusPaused,
		)
	}

	if len(snapshot.ActiveFaults) != 1 {
		t.Fatalf(
			"active faults = %d, want 1",
			len(snapshot.ActiveFaults),
		)
	}

	if snapshot.ActiveFaults[0].Component != fault.ComponentNavigationSensor {
		t.Errorf(
			"component = %q, want %q",
			snapshot.ActiveFaults[0].Component,
			fault.ComponentNavigationSensor,
		)
	}

	if snapshot.ActiveFaults[0].Status != fault.StatusActive {
		t.Errorf(
			"fault status = %q, want %q",
			snapshot.ActiveFaults[0].Status,
			fault.StatusActive,
		)
	}

	// Injecting the same active failure must be rejected.
	response = performFaultRequest(
		t,
		server,
		"/api/faults/inject",
		`{"component":"navigation_sensor"}`,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"duplicate injection status = %d, want %d",
			response.Code,
			http.StatusConflict,
		)
	}

	// Resuming is prohibited while the sensor failure remains active.
	response = performFaultRequest(
		t,
		server,
		"/api/mission/resume",
		"",
	)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"resume with active fault status = %d, want %d",
			response.Code,
			http.StatusConflict,
		)
	}

	// Recover the failed sensor.
	response = performFaultRequest(
		t,
		server,
		"/api/faults/recover",
		`{"component":"navigation_sensor"}`,
	)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"recovery status = %d, want %d; body: %s",
			response.Code,
			http.StatusOK,
			response.Body.String(),
		)
	}

	snapshot = decodeFaultSnapshot(t, response)

	if snapshot.Status != simulation.StatusPaused {
		t.Errorf(
			"status after recovery = %q, want %q",
			snapshot.Status,
			simulation.StatusPaused,
		)
	}

	if len(snapshot.ActiveFaults) != 0 {
		t.Errorf(
			"active faults after recovery = %d, want 0",
			len(snapshot.ActiveFaults),
		)
	}

	if len(snapshot.FaultHistory) != 1 {
		t.Fatalf(
			"fault history = %d, want 1",
			len(snapshot.FaultHistory),
		)
	}

	if snapshot.FaultHistory[0].Status != fault.StatusRecovered {
		t.Errorf(
			"history status = %q, want %q",
			snapshot.FaultHistory[0].Status,
			fault.StatusRecovered,
		)
	}

	// Recovery does not automatically resume the mission.
	response = performFaultRequest(
		t,
		server,
		"/api/mission/resume",
		"",
	)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"resume after recovery status = %d, want %d; body: %s",
			response.Code,
			http.StatusOK,
			response.Body.String(),
		)
	}

	snapshot = decodeFaultSnapshot(t, response)

	if snapshot.Status != simulation.StatusPlanned {
		t.Errorf(
			"status after resume = %q, want %q",
			snapshot.Status,
			simulation.StatusPlanned,
		)
	}

	if len(snapshot.ActiveFaults) != 0 {
		t.Errorf(
			"active faults after resume = %d, want 0",
			len(snapshot.ActiveFaults),
		)
	}
}

func TestFaultAPIRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed JSON",
			body: `{`,
		},
		{
			name: "unknown component",
			body: `{"component":"unknown_sensor"}`,
		},
		{
			name: "unknown field",
			body: `{"component":"navigation_sensor","extra":true}`,
		},
		{
			name: "multiple JSON values",
			body: `{"component":"navigation_sensor"} {"component":"battery_sensor"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestAPIServer(t)

			response := performFaultRequest(
				t,
				server,
				"/api/faults/inject",
				tt.body,
			)

			if response.Code != http.StatusBadRequest {
				t.Errorf(
					"status = %d, want %d; body: %s",
					response.Code,
					http.StatusBadRequest,
					response.Body.String(),
				)
			}
		})
	}
}

func TestFaultAPIRejectsRecoveryWithoutActiveFault(t *testing.T) {
	server := newTestAPIServer(t)

	response := performFaultRequest(
		t,
		server,
		"/api/faults/recover",
		`{"component":"navigation_sensor"}`,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d; body: %s",
			response.Code,
			http.StatusConflict,
			response.Body.String(),
		)
	}
}

func TestFaultAPIRejectsWrongHTTPMethod(t *testing.T) {
	server := newTestAPIServer(t)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/faults/inject",
		nil,
	)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"status = %d, want %d",
			response.Code,
			http.StatusMethodNotAllowed,
		)
	}
}
