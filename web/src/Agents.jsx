import TopNav from "./TopNav.jsx";

export default function Agents({ role }) {
  const [agents, setAgents] = React.useState([]);
  const [displayName, setDisplayName] = React.useState("");
  const [issuedToken, setIssuedToken] = React.useState(null);
  const [error, setError] = React.useState(null);

  function loadAgents() {
    setError(null);
    fetch("/api/agents")
      .then((res) => res.json())
      .then(setAgents)
      .catch((err) => setError(String(err)));
  }

  React.useEffect(() => {
    loadAgents();
  }, []);

  function handleCreate(e) {
    e.preventDefault();
    fetch("/api/agents", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ display_name: displayName }),
    })
      .then((res) => res.json())
      .then(() => {
        setDisplayName("");
        loadAgents();
      })
      .catch((err) => setError(String(err)));
  }

  function handleIssueToken(agentID) {
    fetch(`/api/agents/${agentID}/tokens`, { method: "POST" })
      .then((res) => res.json())
      .then((data) => setIssuedToken(data.token))
      .catch((err) => setError(String(err)));
  }

  function handleRevoke(agentID) {
    fetch(`/api/agents/${agentID}`, { method: "DELETE" })
      .then(loadAgents)
      .catch((err) => setError(String(err)));
  }

  const activeCount = agents.filter((agent) => agent.has_active_token).length;

  return (
    <>
      <header className="site">
        <h1>AI Rendezvous Point</h1>
        <TopNav active="agents" role={role} />
        <span className="spacer"></span>
        <span className="count">
          {activeCount} signaling / {agents.length} registered
        </span>
      </header>
      <main>
        <h2 className="section-title">Agents</h2>
        {error && <div className="callout">Couldn't reach the server: {error}</div>}
        {issuedToken && (
          <div className="transmission">
            <span className="label">New token</span>
            <code>{issuedToken}</code>
            <button onClick={() => setIssuedToken(null)}>Copied, dismiss</button>
          </div>
        )}
        <form className="dispatch-bar" onSubmit={handleCreate}>
          <input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Name a new agent"
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
                  <a className="name" href={"#/profiles/" + agent.id}>
                    {agent.display_name}
                  </a>
                  <span className={"status-label" + (agent.has_active_token ? " active" : "")}>
                    {agent.has_active_token ? "signaling" : "revoked"}
                  </span>
                </div>
                <code className="agent-id">{agent.id}</code>
                <div className="actions">
                  <button onClick={() => handleIssueToken(agent.id)}>Issue token</button>
                  <button onClick={() => handleRevoke(agent.id)}>Revoke tokens</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </>
  );
}
