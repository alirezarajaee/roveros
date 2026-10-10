
import { useEffect, useState } from "react";

const API_BASE = "http://127.0.0.1:8081";

type SensorComponent =
  | "navigation_sensor"
  | "obstacle_sensor"
  | "battery_sensor";

interface FaultEvent {
  id: number;
  type: string;
  component: SensorComponent;
  status: "active" | "recovered" | string;
  message: string;
}

interface FaultSnapshot {
  status: string;
  safety_reason?: string;
  active_faults?: FaultEvent[];
  fault_history?: FaultEvent[];
}

interface FaultPanelProps {
  onStateRefresh: () => Promise<void>;
}

const SENSOR_LABELS: Record<SensorComponent, string> = {
  navigation_sensor: "Navigation sensor",
  obstacle_sensor: "Obstacle sensor",
  battery_sensor: "Battery sensor",
};

export default function FaultPanel({
  onStateRefresh,
}: FaultPanelProps) {
  const [snapshot, setSnapshot] = useState<FaultSnapshot | null>(null);
  const [component, setComponent] =
    useState<SensorComponent>("navigation_sensor");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [loadError, setLoadError] = useState("");

  // Keep fault information synchronized with the Go backend.
  useEffect(() => {
    let cancelled = false;
    let inFlight = false;

    async function loadState() {
      if (inFlight) return;
      inFlight = true;

      try {
        const response = await fetch(`${API_BASE}/api/state`);

        if (!response.ok) {
          throw new Error(`Could not read API state: HTTP ${response.status}`);
        }

        const data = (await response.json()) as FaultSnapshot;

        if (!cancelled) {
          setSnapshot(data);
          setLoadError("");
        }
      } catch (error) {
        if (!cancelled) {
          setLoadError(
            error instanceof Error
              ? error.message
              : "Could not load fault status",
          );
        }
      } finally {
        inFlight = false;
      }
    }

    void loadState();
    const timer = window.setInterval(() => void loadState(), 1000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  const activeFaults = snapshot?.active_faults ?? [];
  const faultHistory = snapshot?.fault_history ?? [];

  const selectedFaultIsActive = activeFaults.some(
    (fault) => fault.component === component,
  );

  async function runFaultAction(action: "inject" | "recover") {
    if (busy) return;

    setBusy(true);
    setNotice("");

    try {
      const response = await fetch(`${API_BASE}/api/faults/${action}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ component }),
      });

      const data = (await response.json()) as
        | FaultSnapshot
        | { error?: string };

      if (!response.ok) {
        const message =
          "error" in data && data.error
            ? data.error
            : `Request failed: HTTP ${response.status}`;

        throw new Error(message);
      }

      const updatedSnapshot = data as FaultSnapshot;
      setSnapshot(updatedSnapshot);

      setNotice(
        action === "inject"
          ? `${SENSOR_LABELS[component]} failure injected.`
          : `${SENSOR_LABELS[component]} recovery recorded.`,
      );

      try {
        await onStateRefresh();
      } catch {
        setNotice(
          "Fault operation succeeded, but refreshing the main dashboard failed.",
        );
      }
    } catch (error) {
      setNotice(
        error instanceof Error ? error.message : "Fault operation failed",
      );
    } finally {
      setBusy(false);
    }
  }

  const buttonStyle = (primary = false): React.CSSProperties => ({
    minHeight: 36,
    padding: "9px 12px",
    borderRadius: 6,
    border: primary ? "1px solid #9e613c" : "1px solid #343844",
    background: primary ? "#e89459" : "#191c24",
    color: primary ? "#1d120c" : "#d5d7df",
    fontSize: 10,
    fontWeight: 700,
    cursor: busy ? "wait" : "pointer",
    opacity: busy ? 0.65 : 1,
  });

  return (
    <article className="panel fault-panel">
      <div className="panel-header">
        <div>
          <div className="eyebrow">FAULT MANAGEMENT</div>
          <h2>Sensor faults & recovery</h2>
        </div>
        <span className="panel-tag">SIMULATED</span>
      </div>

      <div style={{ padding: "0 16px 16px" }}>
        <label
          htmlFor="fault-component"
          style={{
            display: "block",
            marginBottom: 8,
            color: "#999fad",
            fontSize: 10,
          }}
        >
          SENSOR COMPONENT
        </label>

        <select
          id="fault-component"
          value={component}
          onChange={(event) =>
            setComponent(event.target.value as SensorComponent)
          }
          disabled={busy}
          style={{
            width: "100%",
            minHeight: 39,
            padding: "8px 10px",
            border: "1px solid #343844",
            borderRadius: 6,
            background: "#171a22",
            color: "#e9ebf1",
            fontSize: 11,
          }}
        >
          <option value="navigation_sensor">Navigation sensor</option>
          <option value="obstacle_sensor">Obstacle sensor</option>
          <option value="battery_sensor">Battery sensor</option>
        </select>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: 9,
            marginTop: 12,
          }}
        >
          <button
            type="button"
            style={buttonStyle(true)}
            disabled={busy || selectedFaultIsActive}
            onClick={() => void runFaultAction("inject")}
          >
            INJECT FAILURE
          </button>

          <button
            type="button"
            style={buttonStyle()}
            disabled={busy || !selectedFaultIsActive}
            onClick={() => void runFaultAction("recover")}
          >
            RECOVER SENSOR
          </button>
        </div>

        {loadError && (
          <div
            role="alert"
            style={{ marginTop: 12, color: "#ffb49b", fontSize: 11 }}
          >
            {loadError}
          </div>
        )}

        {notice && (
          <div
            role="status"
            aria-live="polite"
            style={{
              marginTop: 12,
              color: notice.includes("failed") || notice.includes("Failure")
                ? "#ffb49b"
                : "#8de0ad",
              fontSize: 11,
              overflowWrap: "anywhere",
            }}
          >
            {notice}
          </div>
        )}

        <div
          style={{
            marginTop: 15,
            padding: 11,
            border: "1px solid #34313a",
            borderRadius: 6,
            background: "#15151c",
          }}
        >
          <div className="eyebrow">SAFETY STATE</div>
          <div style={{ marginTop: 7, color: "#dedfe6", fontSize: 11 }}>
            {snapshot?.safety_reason ||
              (activeFaults.length > 0
                ? "Active sensor faults require attention."
                : "No active sensor faults reported.")}
          </div>
        </div>

        <div style={{ marginTop: 18 }}>
          <div className="eyebrow">ACTIVE FAULTS ({activeFaults.length})</div>

          {activeFaults.length === 0 ? (
            <div style={{ marginTop: 9, color: "#858b9b", fontSize: 11 }}>
              No active sensor faults.
            </div>
          ) : (
            <div style={{ display: "grid", gap: 8, marginTop: 9 }}>
              {activeFaults.map((fault) => (
                <div
                  key={fault.id}
                  style={{
                    padding: 10,
                    border: "1px solid #704036",
                    borderRadius: 6,
                    background: "#281916",
                  }}
                >
                  <div style={{ color: "#ffb49b", fontSize: 11, fontWeight: 600 }}>
                    {SENSOR_LABELS[fault.component] ?? fault.component}
                  </div>
                  <div style={{ marginTop: 5, color: "#c6a39a", fontSize: 10 }}>
                    {fault.message}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ marginTop: 18 }}>
          <div className="eyebrow">FAULT HISTORY ({faultHistory.length})</div>

          {faultHistory.length === 0 ? (
            <div style={{ marginTop: 9, color: "#858b9b", fontSize: 11 }}>
              No faults recorded in this mission.
            </div>
          ) : (
            <div style={{ display: "grid", gap: 8, marginTop: 9 }}>
              {[...faultHistory].reverse().slice(0, 5).map((fault) => (
                <div
                  key={fault.id}
                  style={{
                    display: "flex",
                    alignItems: "start",
                    justifyContent: "space-between",
                    gap: 10,
                    padding: 9,
                    border: "1px solid #2d303a",
                    borderRadius: 6,
                    background: "#14171e",
                  }}
                >
                  <div>
                    <div style={{ color: "#dfe1e8", fontSize: 10 }}>
                      {SENSOR_LABELS[fault.component] ?? fault.component}
                    </div>
                    <div style={{ marginTop: 4, color: "#858b9b", fontSize: 9 }}>
                      Event #{fault.id}
                    </div>
                  </div>
                  <span
                    style={{
                      color: fault.status === "active" ? "#ffb49b" : "#8de0ad",
                      fontSize: 9,
                      fontFamily: "monospace",
                    }}
                  >
                    {fault.status.toUpperCase()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </article>
  );
}
