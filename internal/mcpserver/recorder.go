package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/auth"
	"github.com/JetManiack/mcp-weather/internal/storage"
)

const DefaultPreviewBytes = 64 << 10 // 64 KiB

// Recorder writes a storage.ToolCall row per MCP tool invocation.
// A zero Recorder (nil DB) records nothing.
type Recorder struct {
	DB           *gorm.DB
	PreviewBytes int
}

func (rec Recorder) previewLimit() int {
	if rec.PreviewBytes <= 0 {
		return DefaultPreviewBytes
	}
	return rec.PreviewBytes
}

// Recorded wraps a tool handler so every call is persisted to the audit log.
// Recording cannot fail the call: a database error is logged and the agent
// still gets its answer. Observability that breaks what it observes is worse
// than no observability.
func Recorded[In, Out any](rec Recorder, tool string, h mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()
		result, out, err := h(ctx, req, in)
		rec.record(ctx, tool, in, out, err, time.Since(start))
		return result, out, err
	}
}

// record persists one call. Context is intentionally NOT passed to the
// database: by the time a slow tool returns, the client may have
// disconnected and cancelled it, and that would drop exactly the calls
// most worth having in the audit log.
func (rec Recorder) record(ctx context.Context, tool string, in, out any, callErr error, elapsed time.Duration) {
	if rec.DB == nil {
		return
	}
	actor, ok := auth.ActorFromContext(ctx)
	if !ok {
		slog.Warn("tool call not recorded: no authenticated actor in context", "tool", tool)
		return
	}

	call := &storage.ToolCall{
		ActorID:    actor.ID,
		Tool:       tool,
		InputJSON:  marshalForHistory(in),
		DurationMS: int(elapsed.Milliseconds()),
	}
	if callErr != nil {
		call.IsError = true
		call.OutputJSON = marshalForHistory(struct {
			Error string `json:"error"`
		}{Error: callErr.Error()})
	} else {
		encoded := marshalForHistory(out)
		call.OutputSize = len(encoded)
		call.OutputJSON, call.Truncated = truncateUTF8(encoded, rec.previewLimit())
	}

	if err := storage.RecordToolCall(rec.DB, call); err != nil {
		slog.Error("failed to record tool call", "tool", tool, "actor_id", actor.ID, "error", err)
	}
}

func marshalForHistory(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("{%q:%q}", "history_encoding_error", err.Error())
	}
	return string(encoded)
}

func truncateUTF8(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := s[:limit]
	for len(cut) > 0 {
		r, size := utf8.DecodeLastRuneInString(cut)
		if r == utf8.RuneError && size <= 1 {
			cut = cut[:len(cut)-1]
			continue
		}
		break
	}
	return string([]byte(cut)), true
}
