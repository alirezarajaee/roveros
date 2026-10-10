
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import "./App.css";

const API_BASE = "http://127.0.0.1:8081";

interface Point {
  x: number;
  y: number;
}

interface RoverState {
  position: Point;
  battery_percent: number;
  status: string;
}

interface MissionSnapshot {
  rover: RoverState;
  status: string;
  target: Point;
  has_target: boolean;
  path: Point[];
  next_step: number;
  steps: number;
  last_error?: string;
}

const WIDTH = 20;
const HEIGHT = 10;
const DEFAULT_GOAL: Point = { x: 15, y: 7 };
const EMPTY_PATH: Point[] = [];
const ROCKS = new Set([
  "4,2",
  "4,3",
  "4,4",
  "12,4",
  "12,5",
]);

const pointKey = (point: Point) => `${point.x},${point.y}`;

async function responseError(response: Response): Promise<string> {
  try {
    const body = (await response.json()) as { error?: string };
    return body.error ?? `HTTP ${response.status}`;
  } catch {
    return `HTTP ${response.status}`;
  }
}

function Icon({ children }: { children: ReactNode }) {
  return (
    <span className="nav-icon" aria-hidden="true">
      {children}
    </span>
  );
}

function App() {
  const [snapshot, setSnapshot] = useState<MissionSnapshot | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const requestInFlight = useRef(false);

  // Read the initial mission state from the real Go API.
  useEffect(() => {
    let cancelled = false;

    async function loadState() {
      try {
        const response = await fetch(`${API_BASE}/api/state`);

        if (!response.ok) {
          throw new Error(await responseError(response));
        }

        const data = (await response.json()) as MissionSnapshot;

        if (!cancelled) {
          setSnapshot(data);
          setError("");
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Could not connect to ROVEROS API",
          );
        }
      }
    }

    void loadState();

    return () => {
      cancelled = true;
    };
  }, []);

  // Advance the real Go simulation while the mission is running.
  useEffect(() => {
    if (snapshot?.status !== "running") {
      return;
    }

    let cancelled = false;

    const timer = window.setInterval(async () => {
      if (requestInFlight.current) {
        return;
      }

      requestInFlight.current = true;

      try {
        const response = await fetch(`${API_BASE}/api/mission/step`, {
          method: "POST",
        });

        if (!response.ok) {
          // The mission may have been paused or completed during a request.
          // Re-read the canonical state before treating this as a connection error.
          const stateResponse = await fetch(`${API_BASE}/api/state`);

          if (!stateResponse.ok) {
            throw new Error(await responseError(response));
          }

          const latest = (await stateResponse.json()) as MissionSnapshot;

          if (!cancelled) {
            setSnapshot(latest);
            setError("");
          }

          return;
        }

        const data = (await response.json()) as MissionSnapshot;

        if (!cancelled) {
          setSnapshot(data);
          setError("");
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Simulation request failed",
          );
        }
      } finally {
        requestInFlight.current = false;
      }
    }, 450);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [snapshot?.status]);

  async function postCommand(
    endpoint: string,
  ): Promise<MissionSnapshot | null> {
    if (requestInFlight.current) {
      setError("A request is still in progress. Please try again.");
      return null;
    }

    requestInFlight.current = true;
    setBusy(true);

    try {
      const response = await fetch(`${API_BASE}${endpoint}`, {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error(await responseError(response));
      }

      const data = (await response.json()) as MissionSnapshot;
      setSnapshot(data);
      setError("");
      return data;
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Command failed",
      );
      return null;
    } finally {
      requestInFlight.current = false;
      setBusy(false);
    }
  }

  async function handlePrimaryAction() {
    if (!snapshot || busy) {
      return;
    }

    switch (snapshot.status) {
      case "running":
        await postCommand("/api/mission/pause");
        break;

      case "paused":
        await postCommand("/api/mission/resume");
        break;

      case "completed":
      case "failed": {
        const reset = await postCommand("/api/mission/reset");

        if (reset) {
          await postCommand("/api/mission/start");
        }
        break;
      }

      case "planned":
        await postCommand("/api/mission/start");
        break;

      default:
        setError(`Cannot start a mission with status "${snapshot.status}".`);
    }
  }

  const currentPosition = snapshot?.rover.position ?? { x: -1, y: -1 };
  const target = snapshot?.target ?? DEFAULT_GOAL;
  const path = snapshot?.path ?? EMPTY_PATH;
  const steps = snapshot?.steps ?? 0;
  const totalMoves = Math.max(path.length - 1, 1);
  const battery = snapshot?.rover.battery_percent ?? 0;
  const progress = snapshot
    ? Math.min(100, Math.round((steps / totalMoves) * 100))
    : 0;

  const completed = snapshot?.status === "completed";
  const running = snapshot?.status === "running";
  const paused = snapshot?.status === "paused";

  const statusLabel = snapshot
    ? snapshot.status === "planned"
      ? "READY"
      : snapshot.status.toUpperCase()
    : "CONNECTING";

  const pathCells = useMemo(
    () => new Set(path.map(pointKey)),
    [path],
  );

  function renderTerrainCell(x: number, y: number) {
    const point = { x, y };
    const key = pointKey(point);

    const isRover = pointKey(currentPosition) === key;
    const isGoal = pointKey(target) === key;
    const isWall = x === 8 && y < 8;
    const isRock = ROCKS.has(key);
    const isObstacle = isWall || isRock;
    const isPath = pathCells.has(key);

    const classes = [
      "terrain-cell",
      isObstacle ? "obstacle" : "",
      isPath ? "path-cell" : "",
      isRover ? "rover-cell" : "",
      isGoal ? "goal-cell" : "",
      isRover && isGoal ? "rover-at-goal" : "",
    ]
      .filter(Boolean)
      .join(" ");

    return (
      <div
        className={classes}
        key={key}
        title={`Grid coordinate (${x}, ${y})`}
      >
        {isRover && isGoal
          ? "X"
          : isRover
            ? "R"
            : isGoal
              ? "G"
              : isObstacle
                ? isRock
                  ? "▪"
                  : ""
                : isPath
                  ? "·"
                  : ""}
      </div>
    );
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-symbol">R</div>
          <div>
            <div className="brand-name">ROVEROS</div>
            <div className="brand-subtitle">AUTONOMOUS SYSTEMS</div>
          </div>
        </div>

        <div className="sidebar-label">MISSION OPERATIONS</div>

        <div className="nav-item active">
          <Icon>◫</Icon>
          <span>Mission overview</span>
        </div>
        <div className="nav-item">
          <Icon>⌖</Icon>
          <span>Terrain & navigation</span>
        </div>
        <div className="nav-item">
          <Icon>⌁</Icon>
          <span>Rover telemetry</span>
        </div>
        <div className="nav-item">
          <Icon>⚙</Icon>
          <span>Subsystem status</span>
        </div>
        <div className="nav-item">
          <Icon>≡</Icon>
          <span>Mission events</span>
        </div>

        <div className="sidebar-bottom">
          <div className="connection-indicator">
            <span className={`status-dot ${snapshot ? "green" : ""}`} />
            <span>{snapshot ? "GO API CONNECTED" : "CONNECTING TO API"}</span>
          </div>
          <div className="sidebar-version">ROVEROS / DEV BUILD</div>
        </div>
      </aside>

      <main className="main-content">
        <header className="topbar">
          <div className="breadcrumb">
            OPERATIONS <span>/</span> MISSION CONTROL
          </div>

          <div className="topbar-right">
            <span className="demo-badge">
              {snapshot ? "GO BACKEND CONNECTED" : "CONNECTING"}
            </span>
            <div className="avatar">RO</div>
          </div>
        </header>

        <section className="page-heading">
          <div>
            <div className="eyebrow">AUTONOMOUS EXPLORATION PROGRAM</div>
            <h1>Mission overview</h1>
            <p>
              Monitor rover navigation, mission progress, and onboard systems.
            </p>
          </div>

          <div className="mission-identity">
            <span className="mission-id">MISSION / RV-001</span>
            <span className={`mission-status ${(snapshot?.status ?? "idle").toLowerCase()}`}>
              <span className="status-dot" />
              {statusLabel}
            </span>
          </div>
        </section>

        {error && (
          <div
            role="alert"
            style={{
              marginBottom: 16,
              padding: "12px 14px",
              border: "1px solid #8b493d",
              borderRadius: 7,
              background: "#291815",
              color: "#ffb49b",
              fontSize: 12,
              overflowWrap: "anywhere",
            }}
          >
            API notice: {error}
          </div>
        )}

        <section className="metrics-grid">
          <article className="metric-card">
            <div className="metric-top">
              <span>MISSION PROGRESS</span>
              <span className="metric-symbol">↗</span>
            </div>
            <div className="metric-value">
              {snapshot ? progress : "—"}<span>{snapshot ? "%" : ""}</span>
            </div>
            <div className="progress-track">
              <div
                className="progress-fill"
                style={{ width: `${progress}%` }}
              />
            </div>
            <div className="metric-foot">
              STEP {steps.toString().padStart(2, "0")} / {path.length ? totalMoves : "—"}
            </div>
          </article>

          <article className="metric-card">
            <div className="metric-top">
              <span>BATTERY LEVEL</span>
              <span className="metric-symbol">ϟ</span>
            </div>
            <div className="metric-value">
              {snapshot ? Math.round(battery) : "—"}<span>{snapshot ? "%" : ""}</span>
            </div>
            <div className="battery-track">
              <div
                className={`battery-fill ${battery < 25 ? "low" : ""}`}
                style={{ width: `${snapshot ? battery : 0}%` }}
              />
            </div>
            <div className="metric-foot">GO SIMULATION TELEMETRY</div>
          </article>

          <article className="metric-card">
            <div className="metric-top">
              <span>TRAVERSED CELLS</span>
              <span className="metric-symbol">⌁</span>
            </div>
            <div className="metric-value">
              {snapshot ? steps : "—"}<span>{snapshot ? " cells" : ""}</span>
            </div>
            <div className="metric-foot">GRID-BASED DISTANCE</div>
          </article>

          <article className="metric-card">
            <div className="metric-top">
              <span>ROVER POSITION</span>
              <span className="metric-symbol">⌖</span>
            </div>
            <div className="coordinate-value">
              {snapshot ? currentPosition.x.toString().padStart(2, "0") : "—"}
              <span>,</span>
              {snapshot ? currentPosition.y.toString().padStart(2, "0") : "—"}
            </div>
            <div className="metric-foot">GRID COORDINATES · X / Y</div>
          </article>
        </section>

        <section className="workspace-grid">
          <article className="panel terrain-panel">
            <div className="panel-header">
              <div>
                <div className="eyebrow">LIVE SIMULATION VIEW</div>
                <h2>Terrain & navigation</h2>
              </div>
              <span className="panel-tag">2D / GRID</span>
            </div>

            <div className="map-meta">
              <span>
                <span className={`status-dot ${snapshot ? "green" : ""}`} />
                {snapshot ? "GO STATE SYNCHRONIZED" : "WAITING FOR BACKEND"}
              </span>
              <span>20 × 10 GRID</span>
            </div>

            <div className="terrain-viewport">
              <div
                className="terrain-grid"
                style={{
                  gridTemplateColumns: `repeat(${WIDTH}, minmax(0, 1fr))`,
                }}
              >
                {Array.from({ length: HEIGHT }, (_, y) =>
                  Array.from({ length: WIDTH }, (_, x) =>
                    renderTerrainCell(x, y),
                  ),
                )}
              </div>
            </div>

            <div className="map-legend">
              <span><i className="legend-rover">R</i> Rover</span>
              <span><i className="legend-goal">G</i> Destination</span>
              <span><i className="legend-obstacle" /> Obstacle</span>
              <span><i className="legend-path" /> Planned path</span>
            </div>

            <div className="coordinates-strip">
              <div>
                <span className="data-label">START POSITION</span>
                <strong>(01, 01)</strong>
              </div>
              <div>
                <span className="data-label">TARGET POSITION</span>
                <strong>
                  ({target.x.toString().padStart(2, "0")}, {target.y.toString().padStart(2, "0")})
                </strong>
              </div>
              <div>
                <span className="data-label">PLANNER</span>
                <strong>A* SEARCH</strong>
              </div>
            </div>
          </article>

          <div className="right-column">
            <article className="panel control-panel">
              <div className="panel-header">
                <div>
                  <div className="eyebrow">MISSION EXECUTION</div>
                  <h2>Rover control</h2>
                </div>
                <span className="control-indicator">
                  <span className={`status-dot ${snapshot ? "green" : ""}`} />
                </span>
              </div>

              <div className="rover-visual">
                <div className="rover-orbit orbit-one" />
                <div className="rover-orbit orbit-two" />
                <div className="rover-core">
                  <span>R</span>
                </div>
                <div className="rover-coordinate">
                  {snapshot
                    ? `X ${currentPosition.x.toString().padStart(2, "0")} / Y ${currentPosition.y.toString().padStart(2, "0")}`
                    : "WAITING FOR TELEMETRY"}
                </div>
              </div>

              <div className="control-status">
                <span>MISSION STATE</span>
                <strong>{statusLabel}</strong>
              </div>

              <button
                className="primary-button"
                onClick={() => void handlePrimaryAction()}
                disabled={!snapshot || busy}
              >
                {running
                  ? "Ⅱ  PAUSE SIMULATION"
                  : paused
                    ? "▶  RESUME SIMULATION"
                    : completed || snapshot?.status === "failed"
                      ? "↻  RUN AGAIN"
                      : snapshot?.status === "planned"
                        ? "▶  START SIMULATION"
                        : busy
                          ? "PROCESSING..."
                          : "CONNECTING..."}
              </button>

              <button
                className="secondary-button"
                onClick={() => void postCommand("/api/mission/reset")}
                disabled={!snapshot || busy}
              >
                ↺ &nbsp; RESET MISSION
              </button>

              <div className="control-note">
                <span className="note-dot" />
                {error
                  ? "API request requires attention"
                  : snapshot
                    ? "Controls linked to Go simulation"
                    : "Connecting to Go backend..."}
              </div>
            </article>

            <article className="panel telemetry-panel">
              <div className="panel-header compact">
                <div>
                  <div className="eyebrow">ONBOARD SYSTEMS</div>
                  <h2>Telemetry</h2>
                </div>
                <span className="panel-tag">
                  {snapshot ? "LIVE API" : "WAITING"}
                </span>
              </div>

              <div className="telemetry-row">
                <span className="telemetry-icon">ϟ</span>
                <div className="telemetry-info">
                  <span>Power subsystem</span>
                  <small>Battery charge</small>
                </div>
                <strong>{snapshot ? `${Math.round(battery)}%` : "—"}</strong>
              </div>

              <div className="telemetry-row">
                <span className="telemetry-icon">⌖</span>
                <div className="telemetry-info">
                  <span>Navigation</span>
                  <small>Path planner</small>
                </div>
                <strong className="text-green">
                  {snapshot ? "READY" : "WAITING"}
                </strong>
              </div>

              <div className="telemetry-row">
                <span className="telemetry-icon">⌁</span>
                <div className="telemetry-info">
                  <span>Drive system</span>
                  <small>Mobility state</small>
                </div>
                <strong>
                  {running
                    ? "ACTIVE"
                    : completed
                      ? "IDLE"
                      : paused
                        ? "PAUSED"
                        : snapshot
                          ? "STANDBY"
                          : "—"}
                </strong>
              </div>

              {snapshot?.last_error && (
                <div style={{ padding: "0 16px 14px", color: "#ffb49b", fontSize: 11 }}>
                  {snapshot.last_error}
                </div>
              )}
            </article>
          </div>
        </section>

        <footer className="page-footer">
          <span>ROVEROS AUTONOMOUS ROVER SIMULATOR</span>
          <span>GO SIMULATION ENGINE · LOCAL DEVELOPMENT</span>
        </footer>
      </main>
    </div>
  );
}

export default App;
