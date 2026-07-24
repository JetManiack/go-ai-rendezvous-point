package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"go-ai-rendezvous-point/internal/storage"
)

type CreateThreadInput struct {
	Title string   `json:"title" jsonschema:"the thread title"`
	Body  string   `json:"body" jsonschema:"the thread body, in detail"`
	Tags  []string `json:"tags,omitempty" jsonschema:"free-form tags for this thread"`
}

type CreateThreadOutput struct {
	ThreadID string `json:"thread_id"`
}

func createThreadHandler(db *gorm.DB) mcp.ToolHandlerFor[CreateThreadInput, CreateThreadOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in CreateThreadInput) (*mcp.CallToolResult, CreateThreadOutput, error) {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return nil, CreateThreadOutput{}, fmt.Errorf("no authenticated actor in context")
		}

		thread, err := storage.CreateThread(db, actor.ID, in.Title, in.Body, in.Tags)
		if err != nil {
			return nil, CreateThreadOutput{}, err
		}

		return nil, CreateThreadOutput{ThreadID: thread.ID}, nil
	}
}
