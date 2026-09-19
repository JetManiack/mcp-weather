// Package mcpserver exposes weather tools over MCP via Streamable HTTP transport.
package mcpserver

import (
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/weather"
)

const (
	ServerName      = "mcp-weather"
	fallbackVersion = "dev"
)

// Deps is everything the MCP surface needs.
type Deps struct {
	Client  *weather.Client
	DB      *gorm.DB // nil means no audit recording and no token auth
	Version string
}

// RegisterTools adds every MCP tool this server exposes to server.
func RegisterTools(server *mcp.Server, deps Deps) {
	rec := Recorder{DB: deps.DB}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_current_weather",
		Description: "Get the current weather at a location: temperature, feels-like, humidity, wind speed and direction, precipitation, and a text description. Use for 'what is the weather right now?' questions.",
	}, recorded(rec, "get_current_weather", currentWeatherHandler(deps)))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_forecast",
		Description: "Get a daily weather forecast for a location: temperature range, precipitation, wind speed, and a text description for each day. Default is 7 days (one week); maximum is 16 days. Use for 'what will the weather be?' questions.",
	}, recorded(rec, "get_forecast", forecastHandler(deps)))
}

// NewServer builds the MCP server with every tool registered.
func NewServer(deps Deps) *mcp.Server {
	version := deps.Version
	if version == "" {
		version = fallbackVersion
	}
	server := mcp.NewServer(&mcp.Implementation{Name: ServerName, Version: version}, nil)
	RegisterTools(server, deps)
	return server
}

func validateUnits(units string) error {
	if units != "" && units != "metric" && units != "imperial" {
		return fmt.Errorf("units must be \"metric\" or \"imperial\", got %q", units)
	}
	return nil
}

// NewHTTPHandler builds the /mcp handler using Streamable HTTP transport,
// optionally wrapped in bearer-token authentication when deps.DB is set.
func NewHTTPHandler(deps Deps) http.Handler {
	server := NewServer(deps)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	if deps.DB != nil {
		return RequireAgentToken(deps.DB, mcpHandler)
	}
	return mcpHandler
}
