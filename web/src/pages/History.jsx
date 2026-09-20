import { useState, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { formatDuration, formatTime, prettyJSON } from "../format.js";
import { apiCall } from "../api.js";

// summarizeInput renders a call's stored input JSON as one compact line.
function summarizeInput(raw) {
  try {
    const args = JSON.parse(raw);
    if (args && typeof args === "object") {
      const parts = Object.entries(args)
        .filter(([, value]) => value !== "" && value !== 0 && value !== null)
        .map(
          ([key, value]) =>
            `${key}=${typeof value === "string" ? value : JSON.stringify(value)}`
        );
      if (parts.length > 0) return parts.join("  ");
    }
  } catch (err) {
    // Fall through to the raw string below.
  }
  return raw || "—";
}

export default function History() {
  const [calls, setCalls] = useState([]);
  const [actorMap, setActorMap] = useState({});
  const [actorList, setActorList] = useState([]);
  const [nextCursor, setNextCursor] = useState("");
  const [tools, setTools] = useState([]);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState({});

  const [searchParams, setSearchParams] = useSearchParams();
  const tool = searchParams.get("tool") || "";
  const status = searchParams.get("status") || "";  // "ok", "error", or ""
  const actor = searchParams.get("actor") || "";

  function buildURL(cursor) {
    const params = new URLSearchParams();
    if (tool) params.set("tool", tool);
    if (status) params.set("status", status);
    if (actor) params.set("actor", actor);
    if (cursor) params.set("cursor", cursor);
    return "/tool-calls?" + params.toString();
  }

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    apiCall(buildURL(""))
      .then((data) => {
        if (cancelled) return;
        setCalls(data.calls || []);
        setActorMap(data.actors || {});
        setNextCursor(data.next_cursor || "");
      })
      .catch((err) => !cancelled && setError(String(err)))
      .finally(() => !cancelled && setLoading(false));

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tool, status, actor]);

  useEffect(() => {
    apiCall("/tool-calls/tools")
      .then((data) => setTools(data || []))
      .catch(() => setTools([]));

    apiCall("/actors")
      .then((data) => setActorList(data || []))
      .catch(() => setActorList([]));
  }, []);

  function loadMore() {
    apiCall(buildURL(nextCursor))
      .then((data) => {
        setCalls((prev) => prev.concat(data.calls || []));
        setActorMap((prev) => ({ ...prev, ...(data.actors || {}) }));
        setNextCursor(data.next_cursor || "");
      })
      .catch((err) => setError(String(err)));
  }

  function setFilter(key, value) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) {
        next.set(key, value);
      } else {
        next.delete(key);
      }
      return next;
    });
  }

  function toggleExpanded(id) {
    setExpanded((prev) => ({ ...prev, [id]: !prev[id] }));
  }

  return (
    <main>
      <div className="page-header">
        <h2 className="section-title">Tool call history</h2>
        <span className="count">{calls.length} calls shown</span>
      </div>
      {error && <div className="callout">Could not reach the server: {error}</div>}

      <div className="filter-bar">
        <select
          value={actor}
          onChange={(e) => setFilter("actor", e.target.value)}
          aria-label="Filter by agent"
        >
          <option value="">All agents</option>
          {actorList.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
        <select
          value={tool}
          onChange={(e) => setFilter("tool", e.target.value)}
          aria-label="Filter by tool"
        >
          <option value="">All tools</option>
          {tools.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <select
          value={status}
          onChange={(e) => setFilter("status", e.target.value)}
          aria-label="Filter by outcome"
        >
          <option value="">All statuses</option>
          <option value="ok">ok</option>
          <option value="error">error</option>
        </select>
        <span className="grow"></span>
      </div>

      {loading ? (
        <div className="empty-state">Loading…</div>
      ) : calls.length === 0 ? (
        <div className="empty-state">
          No tool calls recorded yet. Point an agent at <code>/mcp</code> and its calls will show
          up here.
        </div>
      ) : (
        <>
          <div className="call-list">
            {calls.map((call) => {
              const statusClass = call.is_error ? "error" : "ok";
              const statusLabel = call.is_error ? "error" : "ok";
              const isExpanded = !!expanded[call.id];
              return (
                <div
                  className={"call-row " + statusClass}
                  key={call.id}
                  onClick={() => toggleExpanded(call.id)}
                  style={{ cursor: "pointer", display: "block", padding: "0.6rem 0.9rem" }}
                >
                  <div style={{ display: "grid", gridTemplateColumns: "4rem 5rem minmax(0,1fr) 7rem 4.5rem 10rem", alignItems: "center", gap: "0.7rem" }}>
                    <span className={"status-badge " + statusClass}>{statusLabel}</span>
                    <span className="tool-badge">{call.tool}</span>
                    <span className="args" title={call.input_json}>
                      {summarizeInput(call.input_json)}
                    </span>
                    <span className="metric" title={actorMap[call.actor_id] || call.actor_id}>
                      {actorMap[call.actor_id] || call.actor_id}
                    </span>
                    <span className="metric">{formatDuration(call.duration_ms)}</span>
                    <time className="metric" dateTime={call.called_at}>
                      {formatTime(call.called_at)}
                    </time>
                  </div>
                  {isExpanded && (
                    <div style={{ marginTop: "0.6rem", display: "flex", flexDirection: "column", gap: "0.4rem" }}>
                      {call.input_json && (
                        <pre className="payload" style={{ maxHeight: "10rem" }}>
                          {prettyJSON(call.input_json)}
                        </pre>
                      )}
                      {call.output_json && (
                        <pre className="payload" style={{ maxHeight: "10rem" }}>
                          {prettyJSON(call.output_json)}
                        </pre>
                      )}
                      {call.truncated && (
                        <span className="truncation-note">Output was truncated.</span>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
          {nextCursor && (
            <button type="button" className="load-more" onClick={loadMore}>
              Load more
            </button>
          )}
        </>
      )}
    </main>
  );
}
