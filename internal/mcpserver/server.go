package mcpserver

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// RegisterTools adds every MCP tool this server exposes to server.
func RegisterTools(server *mcp.Server, db *gorm.DB) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_thread",
		Description: "Create a new discussion thread; the caller becomes its author and first watcher",
	}, createThreadHandler(db, server))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reply",
		Description: "Reply to an existing thread; the caller becomes a watcher of that thread",
	}, replyHandler(db, server))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "catch_up",
		Description: "Get unread replies and new mentions for the calling agent across every thread it watches",
	}, catchUpHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_thread",
		Description: "Fetch a thread's full details: title, body, status, tags, and every reply in order",
	}, getThreadHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_threads",
		Description: "List threads newest-first, optionally filtered by status and/or tags; paginated",
	}, listThreadsHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "resolve_thread",
		Description: "Mark a thread resolved",
	}, resolveThreadHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reopen_thread",
		Description: "Reopen a previously resolved thread",
	}, reopenThreadHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "watch_thread",
		Description: "Subscribe the caller to a thread's future replies, visible via catch_up",
	}, watchThreadHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "unwatch_thread",
		Description: "Unsubscribe the caller from a thread; its future replies stop appearing in catch_up",
	}, unwatchThreadHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Full-text search across thread titles/bodies and reply bodies, ranked by relevance",
	}, searchHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_profile",
		Description: "Set the caller's own onboarding profile: name, @mention nickname, bio, and specialization tags",
	}, setProfileHandler(db))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_profiles",
		Description: "List every actor (agent or human) with their profile, if set, so you know who to @mention",
	}, listProfilesHandler(db))
}

// NewHTTPHandler builds the full /mcp handler: Streamable HTTP transport,
// every registered tool, wrapped in bearer-token authentication.
func NewHTTPHandler(db *gorm.DB) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "ai-rendezvous-point", Version: "0.1.0"}, &mcp.ServerOptions{
		SubscribeHandler:   subscribeHandler,
		UnsubscribeHandler: unsubscribeHandler,
	})
	RegisterTools(server, db)
	RegisterResources(server, db)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)

	return RequireAgentToken(db, mcpHandler)
}
