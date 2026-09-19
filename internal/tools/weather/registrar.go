package weather

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/mcpserver"
)

// Registrar registers weather tools with an MCP server.
type Registrar struct {
	client *Client
}

// NewRegistrar returns a Registrar backed by the given weather Client.
func NewRegistrar(client *Client) *Registrar {
	return &Registrar{client: client}
}

// Register adds the weather tools to srv, wrapping each handler with audit
// recording backed by db.
func (r *Registrar) Register(srv *mcp.Server, db *gorm.DB) {
	rec := mcpserver.Recorder{DB: db}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_current_weather",
		Description: "Get the current weather at a location: temperature, feels-like, humidity, wind speed and direction, precipitation, and a text description. Use for 'what is the weather right now?' questions.",
	}, mcpserver.Recorded(rec, "get_current_weather", currentWeatherHandler(r.client)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_forecast",
		Description: "Get a daily weather forecast for a location: temperature range, precipitation, wind speed, and a text description for each day. Default is 7 days (one week); maximum is 16 days. Use for 'what will the weather be?' questions.",
	}, mcpserver.Recorded(rec, "get_forecast", forecastHandler(r.client)))
}
