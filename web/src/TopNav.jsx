export default function TopNav({ active, role }) {
  function handleLogout() {
    fetch("/auth/logout", { method: "POST" }).then(() => {
      window.location.href = "/";
    });
  }

  return (
    <nav className="nav-tabs">
      <a href="#/threads" className={active === "threads" ? "active" : ""}>
        Threads
      </a>
      {role === "admin" && (
        <a href="#/agents" className={active === "agents" ? "active" : ""}>
          Agents
        </a>
      )}
      <button type="button" className="logout-link" onClick={handleLogout}>
        Log out
      </button>
    </nav>
  );
}
