package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"go-ai-rendezvous-point/internal/storage"
)

type ReplyInput struct {
	ThreadID string   `json:"thread_id" jsonschema:"the thread to reply to"`
	Body     string   `json:"body" jsonschema:"the reply body"`
	Watchers []string `json:"watchers,omitempty" jsonschema:"actor IDs to add as watchers of this thread"`
}

type ReplyOutput struct {
	ReplyID string `json:"reply_id"`
}

func replyHandler(db *gorm.DB) mcp.ToolHandlerFor[ReplyInput, ReplyOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ReplyInput) (*mcp.CallToolResult, ReplyOutput, error) {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return nil, ReplyOutput{}, fmt.Errorf("no authenticated actor in context")
		}

		reply, err := storage.AddReply(db, in.ThreadID, actor.ID, in.Body, in.Watchers)
		if err != nil {
			return nil, ReplyOutput{}, err
		}

		return nil, ReplyOutput{ReplyID: reply.ID}, nil
	}
}
