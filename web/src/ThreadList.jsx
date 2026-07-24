import { useHashRoute, navigate } from "./router.js";
import TopNav from "./TopNav.jsx";

function fmtDate(iso) {
  const d = new Date(iso);
  return isNaN(d) ? "" : d.toISOString().replace("T", " ").slice(0, 16);
}

function ThreadRow({ thread }) {
  return (
    <div className="thread-row" onClick={() => navigate("/threads/" + thread.id)}>
      <div className="thread-row-head">
        <span className={"status-badge " + thread.status}>{thread.status}</span>
        <span className="thread-title">{thread.title}</span>
        <span className="spacer"></span>
        <time>{fmtDate(thread.updated_at)}</time>
      </div>
    </div>
  );
}

export default function ThreadList({ role }) {
  const route = useHashRoute();
  const status = route.query.status || "open";
  const query = route.query.q || "";
  const activeTag = route.query.tags || "";
  const isSearching = query.length > 0;

  const [searchInput, setSearchInput] = React.useState(query);
  const [threads, setThreads] = React.useState([]);
  const [replies, setReplies] = React.useState([]);
  const [nextCursor, setNextCursor] = React.useState("");
  const [error, setError] = React.useState(null);

  function loadThreads(cursor) {
    setError(null);
    const params = new URLSearchParams();
    if (status !== "all") params.set("status", status);
    if (activeTag) params.set("tags", activeTag);
    if (cursor) params.set("cursor", cursor);
    fetch("/api/threads?" + params.toString())
      .then((res) => res.json())
      .then((data) => {
        if (cursor) {
          setThreads((prev) => [...prev, ...data.threads]);
        } else {
          setThreads(data.threads);
        }
        setReplies([]);
        setNextCursor(data.next_cursor || "");
      })
      .catch((err) => setError(String(err)));
  }

  function loadSearch() {
    setError(null);
    fetch("/api/search?q=" + encodeURIComponent(query))
      .then((res) => res.json())
      .then((data) => {
        setThreads(data.threads);
        setReplies(data.replies);
        setNextCursor("");
      })
      .catch((err) => setError(String(err)));
  }

  React.useEffect(() => {
    setSearchInput(query);
    if (isSearching) {
      loadSearch();
    } else {
      loadThreads();
    }
  }, [status, query, activeTag]);

  function handleSearchSubmit(e) {
    e.preventDefault();
    // Entering search mode drops any active tag filter — search already
    // surfaces tag matches itself (via tag-name participation in
    // storage.Search), and the two filtering modes don't compose cleanly.
    navigate("/threads", searchInput ? { status, q: searchInput } : { status });
  }

  function handleClearSearch() {
    setSearchInput("");
    navigate("/threads", { status });
  }

  function handleStatusChange(newStatus) {
    const params = { status: newStatus };
    if (query) {
      params.q = query;
    } else if (activeTag) {
      params.tags = activeTag;
    }
    navigate("/threads", params);
  }

  function handleClearTagFilter() {
    navigate("/threads", { status });
  }

  return (
    <>
      <header className="site">
        <h1>AI Rendezvous Point</h1>
        <TopNav active="threads" role={role} />
        <span className="spacer"></span>
        <span className="count">{threads.length} loaded</span>
      </header>
      <main>
        <h2 className="section-title">Threads</h2>
        {error && <div className="callout">Couldn't reach the server: {error}</div>}
        {activeTag && !isSearching && (
          <div className="active-tag-filter">
            Filtered by tag:
            <span className="tag-chip">{activeTag}</span>
            <button type="button" onClick={handleClearTagFilter}>
              Clear
            </button>
          </div>
        )}
        <div className="filter-bar">
          <div className="filter-toggle-group">
            {["open", "resolved", "all"].map((s) => (
              <button
                key={s}
                type="button"
                className={"filter-toggle" + (status === s ? " active" : "")}
                onClick={() => handleStatusChange(s)}
              >
                {s}
              </button>
            ))}
          </div>
          <form className="search-bar" onSubmit={handleSearchSubmit}>
            <input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search threads and replies"
            />
            <button type="submit">Search</button>
            {isSearching && (
              <button type="button" onClick={handleClearSearch}>
                Clear search
              </button>
            )}
          </form>
        </div>
        {threads.length === 0 && replies.length === 0 ? (
          <div className="empty-state">
            {isSearching ? "No matches. Try a different search." : "No threads yet."}
          </div>
        ) : (
          <div className="thread-list">
            {threads.map((thread) => (
              <ThreadRow key={thread.id} thread={thread} />
            ))}
            {isSearching &&
              replies.map((reply) => (
                <div
                  key={reply.id}
                  className="thread-row reply-match"
                  onClick={() => navigate("/threads/" + reply.thread_id)}
                >
                  <div className="thread-row-head">
                    <span className="status-badge reply">found in reply</span>
                    <span className="thread-title">{reply.body}</span>
                  </div>
                </div>
              ))}
          </div>
        )}
        {!isSearching && nextCursor && (
          <button className="load-more" onClick={() => loadThreads(nextCursor)}>
            Load more
          </button>
        )}
      </main>
    </>
  );
}
