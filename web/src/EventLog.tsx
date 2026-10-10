
export interface MissionEvent {
  id: number;
  type: string;
  level: string;
  message: string;
  timestamp: string;
}

interface EventLogProps {
  events: MissionEvent[];
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);

  if (Number.isNaN(date.getTime())) {
    return "UNKNOWN TIME";
  }

  return date.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

function formatEventType(type: string): string {
  return type.replaceAll("_", " ").toUpperCase();
}

function levelColor(level: string): string {
  switch (level.toLowerCase()) {
    case "error":
      return "#ff8d80";
    case "warning":
      return "#ffbd83";
    default:
      return "#8de0ad";
  }
}

function eventSymbol(type: string): string {
  if (type.includes("fault") && type.includes("recover")) return "✓";
  if (type.includes("fault")) return "!";
  if (type.includes("failed") || type.includes("blocked")) return "×";
  if (type.includes("completed")) return "✓";
  if (type.includes("paused")) return "Ⅱ";
  if (type.includes("resumed") || type.includes("started")) return "▶";
  if (type.includes("moved")) return "⌖";
  if (type.includes("planned")) return "↗";
  return "·";
}

export default function EventLog({ events }: EventLogProps) {
  const recentEvents = [...events].reverse().slice(0, 20);

  return (
    <article className="panel event-log-panel">
      <div className="panel-header">
        <div>
          <div className="eyebrow">MISSION AUDIT TRAIL</div>
          <h2>Mission events</h2>
        </div>

        <span className="panel-tag">
          {events.length} {events.length === 1 ? "EVENT" : "EVENTS"}
        </span>
      </div>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 10,
          padding: "10px 17px",
          borderTop: "1px solid #222630",
          borderBottom: "1px solid #222630",
          background: "#10131a",
          color: "#858b9b",
          font: "9px 'DM Mono', Consolas, monospace",
        }}
      >
        <span>
          <span
            className="status-dot green"
            style={{ marginRight: 8 }}
          />
          ENGINE EVENT STREAM
        </span>

        <span>NEWEST FIRST</span>
      </div>

      {recentEvents.length === 0 ? (
        <div
          style={{
            padding: "27px 18px",
            textAlign: "center",
            color: "#858b9b",
            fontSize: 11,
          }}
        >
          No mission events have been recorded yet.
        </div>
      ) : (
        <div style={{ padding: "4px 17px 10px" }}>
          {recentEvents.map((event, index) => {
            const color = levelColor(event.level);

            return (
              <div
                key={event.id}
                style={{
                  position: "relative",
                  display: "grid",
                  gridTemplateColumns: "28px minmax(0, 1fr)",
                  gap: 11,
                  padding: "13px 0",
                  borderBottom:
                    index === recentEvents.length - 1
                      ? "none"
                      : "1px solid #242833",
                }}
              >
                <div
                  style={{
                    position: "relative",
                    zIndex: 1,
                    width: 27,
                    height: 27,
                    display: "grid",
                    placeItems: "center",
                    border: `1px solid ${color}65`,
                    borderRadius: 7,
                    background: `${color}12`,
                    color,
                    fontSize: 13,
                    fontWeight: 700,
                  }}
                >
                  {eventSymbol(event.type)}
                </div>

                <div style={{ minWidth: 0 }}>
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "space-between",
                      flexWrap: "wrap",
                      gap: 6,
                    }}
                  >
                    <span
                      style={{
                        color: "#dfe1e8",
                        font: "10px 'DM Mono', Consolas, monospace",
                        overflowWrap: "anywhere",
                      }}
                    >
                      {formatEventType(event.type)}
                    </span>

                    <span
                      style={{
                        color: "#858b9b",
                        font: "9px 'DM Mono', Consolas, monospace",
                      }}
                    >
                      {formatTime(event.timestamp)}
                    </span>
                  </div>

                  <p
                    style={{
                      margin: "6px 0 0",
                      color: "#a8adba",
                      fontSize: 11,
                      lineHeight: 1.65,
                      overflowWrap: "anywhere",
                    }}
                  >
                    {event.message}
                  </p>

                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 7,
                      marginTop: 7,
                    }}
                  >
                    <span
                      style={{
                        color,
                        font: "9px 'DM Mono', Consolas, monospace",
                      }}
                    >
                      {event.level.toUpperCase()}
                    </span>

                    <span style={{ color: "#626979", fontSize: 9 }}>
                      · EVENT #{event.id}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {events.length > recentEvents.length && (
        <div
          style={{
            padding: "10px 17px",
            borderTop: "1px solid #242833",
            color: "#858b9b",
            fontSize: 10,
          }}
        >
          Showing the latest {recentEvents.length} events.
        </div>
      )}
    </article>
  );
}
