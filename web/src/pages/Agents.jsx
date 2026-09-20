import { useState, useEffect } from "react";
import { formatTime } from "../format.js";
import { apiCall } from "../api.js";

export default function Agents() {
  const [agents, setAgents] = useState([]);
  const [name, setName] = useState("");
  const [issuedToken, setIssuedToken] = useState(null);
  const [credsByAgent, setCredsByAgent] = useState({});
  const [error, setError] = useState(null);

  function loadAgents() {
    setError(null);
    apiCall("/actors")
      .then(setAgents)
      .catch((err) => setError(String(err)));
  }

  useEffect(loadAgents, []);

  function loadCredentials(agentID) {
    apiCall(`/actors/${agentID}/credentials`)
      .then((creds) => setCredsByAgent((prev) => ({ ...prev, [agentID]: creds })))
      .catch((err) => setError(String(err)));
  }

  function toggleCredentials(agentID) {
    if (credsByAgent[agentID]) {
      setCredsByAgent((prev) => {
        const next = { ...prev };
        delete next[agentID];
        return next;
      });
      return;
    }
    loadCredentials(agentID);
  }

  function handleCreate(e) {
    e.preventDefault();
    apiCall("/actors", {
      method: "POST",
      body: JSON.stringify({ name }),
    })
      .then(() => {
        setName("");
        loadAgents();
      })
      .catch((err) => setError(String(err)));
  }

  function handleIssueToken(agentID) {
    apiCall(`/actors/${agentID}/credentials`, { method: "POST" })
      .then((data) => {
        setIssuedToken(data.token);
        loadAgents();
        if (credsByAgent[agentID]) loadCredentials(agentID);
      })
      .catch((err) => setError(String(err)));
  }

  function handleRevokeToken(credID) {
    apiCall(`/credentials/${credID}`, { method: "DELETE" })
      .then(() => {
        loadAgents();
        for (const agentID of Object.keys(credsByAgent)) {
          loadCredentials(agentID);
        }
      })
      .catch((err) => setError(String(err)));
  }

  function handleRevokeAll(agentID, agentName) {
    if (!confirm(`Revoke all tokens for "${agentName}"? History is preserved.`)) return;
    apiCall(`/actors/${agentID}`, { method: "DELETE" })
      .then(() => {
        loadAgents();
        if (credsByAgent[agentID]) loadCredentials(agentID);
      })
      .catch((err) => setError(String(err)));
  }

  const activeCount = agents.filter((a) => a.has_active_token).length;

  return (
    <main>
      <div className="page-header">
        <h2 className="section-title">Agents</h2>
        <span className="count">{activeCount} active / {agents.length} registered</span>
      </div>
      {error && <div className="callout">{error}</div>}
      {issuedToken && (
        <div className="transmission">
          <span className="label">New token — copy it now</span>
          <code>{issuedToken}</code>
          <button
            type="button"
            onClick={() => {
              navigator.clipboard.writeText(issuedToken);
            }}
          >
            Copy
          </button>
          <button type="button" onClick={() => setIssuedToken(null)}>
            Dismiss
          </button>
        </div>
      )}
      <form className="dispatch-bar" onSubmit={handleCreate}>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Name a new agent"
          aria-label="New agent name"
        />
        <button type="submit" className="primary">
          Register agent
        </button>
      </form>
      {agents.length === 0 ? (
        <div className="empty-state">No agents yet. Register one to issue its first token.</div>
      ) : (
        <div className="agent-grid">
          {agents.map((agent) => (
            <div className="agent-card" key={agent.id}>
              <div className="beacon-row">
                <span className={"beacon" + (agent.has_active_token ? " active" : "")}></span>
                <span className="name">{agent.name}</span>
                <span className={"status-label" + (agent.has_active_token ? " active" : "")}>
                  {agent.has_active_token ? "active" : "no token"}
                </span>
              </div>
              <code className="agent-id">{agent.id}</code>
              <div className="actions">
                <button type="button" onClick={() => handleIssueToken(agent.id)}>
                  Issue token
                </button>
                <button type="button" onClick={() => toggleCredentials(agent.id)}>
                  {credsByAgent[agent.id] ? "Hide credentials" : "Credentials"}
                </button>
                <button type="button" onClick={() => handleRevokeAll(agent.id, agent.name)}>
                  Revoke all
                </button>
              </div>
              {credsByAgent[agent.id] && (
                <ul className="token-list">
                  {credsByAgent[agent.id].length === 0 && <li>No credentials issued.</li>}
                  {credsByAgent[agent.id].map((cred) => (
                    <li key={cred.id}>
                      <span className={cred.revoked_at ? "revoked" : ""}>
                        {cred.id.slice(0, 8)} · {cred.revoked_at ? "revoked" : "active"} ·{" "}
                        {cred.last_used_at
                          ? "used " + formatTime(cred.last_used_at)
                          : "never used"}
                      </span>
                      {!cred.revoked_at && (
                        <button type="button" onClick={() => handleRevokeToken(cred.id)}>
                          Revoke
                        </button>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))}
        </div>
      )}
    </main>
  );
}
