<div align="center">

# ROVEROS

### Rover Simulation & Mission Control Platform

A grid-based autonomous rover simulation featuring A* path planning,
mission execution, simulated telemetry, sensor fault recovery, and
real-time mission event logging.

**Built with Go, React, and TypeScript.**

</div>

---

## Overview

ROVEROS is a software engineering project that simulates the navigation
and operation of a rover in a grid-based environment.

It combines a Go simulation engine and HTTP API with a React dashboard
to make navigation, mission progress, simulated telemetry, and fault
handling observable through a single interface.

The project is designed to demonstrate autonomous navigation concepts,
backend architecture, API integration, state management, and fault-aware
simulation.

> **Project scope:** ROVEROS is currently a software simulation. It does
> not control physical rover hardware or real-world sensors.

## Features

### Navigation and World Simulation

- Grid-based terrain representation with obstacles.
- A* pathfinding for route planning.
- Configurable rover start and destination positions through the
  simulation code.
- Visual representation of the rover, destination, obstacles, and route.

### Mission Control

- Start and execute a planned mission.
- Pause and resume execution.
- Reset the simulation to its initial state.
- Track rover position, mission progress, and simulated battery usage.
- Detect and report mission completion.

### Fault Injection and Recovery

- Inject simulated sensor failures.
- Pause mission execution when the navigation sensor fault is detected.
- Recover from an injected fault.
- Resume mission execution after recovery.
- Track active faults separately from fault history.

### Mission Event Log

- Record mission lifecycle and rover movement events.
- Record sensor fault injection and recovery.
- Include event identifiers and timestamps.
- Retain a bounded in-memory event history.
- Display recent events in the React dashboard.

### Dashboard and API

- React and TypeScript mission-control interface.
- Go HTTP API powering the dashboard.
- Live state refresh during mission execution.
- Dedicated views for navigation, telemetry, subsystem status,
  fault management, and mission events.

## Technology Stack

| Layer | Technology |
|---|---|
| Simulation engine | Go |
| Navigation | A* pathfinding |
| Backend interface | Go HTTP API |
| Frontend | React |
| Language for frontend | TypeScript |
| Frontend tooling | Vite |
| Automated backend tests | Go testing package |
| Frontend quality checks | ESLint and TypeScript build |

## Architecture

The codebase is organized into a Go backend and a separate web frontend.

```text
ROVEROS/
├── cmd/
│   ├── roveros/                  # Console demonstration
│   └── roveros-api/              # HTTP API entry point
├── internal/
│   ├── api/                      # HTTP handlers and API tests
│   ├── fault/                    # Fault injection and recovery
│   ├── mission/                  # Mission event definitions and log
│   ├── navigation/               # A* route planning
│   ├── rover/                    # Rover state and movement
│   ├── simulation/               # Mission simulation engine
│   └── world/                    # Grid and point representation
└── web/
    └── src/
        ├── App.tsx               # Main dashboard
        ├── FaultPanel.tsx        # Fault management interface
        └── EventLog.tsx          # Mission event history
```

### Design principles

- **Separation of concerns:** navigation, simulation, faults, API
  handling, and presentation are implemented in distinct components.
- **Observable execution:** mission state and events are exposed
  through the API and dashboard.
- **Testability:** core Go components include automated tests.
- **Explicit fault handling:** fault state and recovery are represented
  separately from normal mission execution.

## Requirements

Install the following tools:

- [Go](https://go.dev/dl/) — use a version compatible with `go.mod`.
- [Node.js](https://nodejs.org/) and npm.
- [Git](https://git-scm.com/) to clone the repository.

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/alirezarajaee/roveros.git
cd roveros
```

### 2. Run the backend API

From the repository root, start the API:

```bash
go run ./cmd/roveros-api
```

The API listens on:

```text
http://127.0.0.1:8081
```

Keep this terminal open while using the dashboard.

### 3. Start the frontend

Open a second terminal in the repository root:

```bash
cd web
npm install
npm run dev
```

Open the local URL printed by Vite. The default development URL is:

```text
http://localhost:5173
```

The frontend communicates with the Go API on port `8081`.

### 4. Run the console demonstration

To run the separate Go console demonstration, open another terminal
in the repository root and execute:

```bash
go run ./cmd/roveros
```

## Running Tests

### Backend tests

From the repository root:

```bash
go test ./cmd/... ./internal/...
```

### Frontend lint

From the `web` directory:

```bash
npm run lint
```

### Frontend production build

From the `web` directory:

```bash
npm run build
```

## API Reference

The development API currently provides the following endpoints.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/api/health` | Check API health |
| `GET` | `/api/state` | Retrieve the current simulation snapshot |
| `POST` | `/api/mission/start` | Start the mission |
| `POST` | `/api/mission/step` | Advance the mission by one step |
| `POST` | `/api/mission/pause` | Pause mission execution |
| `POST` | `/api/mission/resume` | Resume mission execution |
| `POST` | `/api/mission/reset` | Reset the mission |
| `POST` | `/api/faults/inject` | Inject a simulated fault |
| `POST` | `/api/faults/recover` | Recover a simulated fault |

Fault requests use a JSON component identifier. For example:

```json
{
  "component": "navigation_sensor"
}
```

Use the API routes and payloads implemented by the current server when
integrating other clients.

## Current Simulation Scenario

The default scenario uses a 20 × 10 grid and a rover starting at
`(1, 1)`, with a destination at `(15, 7)`.

The current configured environment includes obstacles and a planned
route. In the verified default run, the rover completed the 22-move
mission with 78% simulated battery remaining.

These values describe the current default scenario, not a benchmark
for real-world robot performance.

## Project Status

The current implementation includes:

- A* navigation and grid-based world simulation.
- Go API integration with a React dashboard.
- Mission lifecycle controls.
- Simulated sensor fault injection and recovery.
- Mission event history.
- Automated Go tests and frontend lint/build checks.

The project is an evolving simulation and development platform.
It has not been validated for physical robot deployment or
safety-critical operation.

## Potential Future Improvements

- Configurable maps and mission scenarios.
- Additional simulated sensor and subsystem failures.
- More extensive navigation and fault-recovery tests.
- Automated frontend-to-backend integration tests.
- API documentation and reproducible demonstration scenarios.
- Screenshots and recorded demonstrations of the dashboard.

## Contributing

Suggestions, bug reports, and technical feedback are welcome.
Please open an issue to discuss a proposed change before submitting
a substantial modification.

## Author

**Alireza Rajaee**

- GitHub: [@alirezarajaee](https://github.com/alirezarajaee)

---

<div align="center">

**ROVEROS — Navigation, simulation, and mission observability.**

</div>