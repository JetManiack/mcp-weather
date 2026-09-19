// Package mcpserver exposes weather tools over MCP via Streamable HTTP transport.
package mcpserver

import (
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/auth"
)

const (
	ServerName      = "mcp-weather"
	fallbackVersion = "dev"
)

// ToolRegistrar registers MCP tools with a server at startup.
type ToolRegistrar interface {
	Register(srv *mcp.Server, db *gorm.DB)
}

// Handler builds the /mcp handler using Streamable HTTP transport.
// Every tool registrar in tools is called to install its tools.
// When db is non-nil the handler is wrapped in bearer-token authentication.
func Handler(db *gorm.DB, tools []ToolRegistrar) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: ServerName, Version: fallbackVersion}, nil)
	for _, t := range tools {
		t.Register(server, db)
	}
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	var h http.Handler = mcpHandler
	if db != nil {
		h = auth.RequireBearer(db, mcpHandler)
	}
	return clearWriteDeadline(h)
}

// clearWriteDeadline removes any write deadline imposed by the HTTP server so
// long-running tool calls (e.g. slow upstream APIs) are not cut mid-response.
func clearWriteDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NewResponseController(w).SetWriteDeadline(time.Time{})
		next.ServeHTTP(w, r)
	})
}
