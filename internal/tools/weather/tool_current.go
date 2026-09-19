package weather

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CurrentInput struct {
	Location string `json:"location" jsonschema:"full official city name in Latin script (e.g. 'London', 'Moscow', 'Saint Petersburg', 'Berlin') or decimal coordinates as 'lat,lon' (e.g. '59.93,30.32'); always use the full official English name, not informal, abbreviated, or non-Latin forms"`
	Units    string `json:"units,omitempty" jsonschema:"temperature units: 'metric' (°C, km/h) or 'imperial' (°F, mph); default is metric"`
}

type CurrentOutput struct {
	Location        string  `json:"location" jsonschema:"resolved location name"`
	Country         string  `json:"country,omitempty" jsonschema:"country name"`
	Time            string  `json:"time" jsonschema:"observation time in local time"`
	Temperature     float64 `json:"temperature" jsonschema:"current temperature"`
	FeelsLike       float64 `json:"feels_like" jsonschema:"apparent (feels-like) temperature"`
	TemperatureUnit string  `json:"temperature_unit" jsonschema:"temperature unit symbol (e.g. °C)"`
	HumidityPercent int     `json:"humidity_percent" jsonschema:"relative humidity in percent"`
	PrecipitationMm float64 `json:"precipitation_mm" jsonschema:"precipitation in the past hour in millimetres"`
	WindSpeed       float64 `json:"wind_speed" jsonschema:"wind speed"`
	WindDirectionDeg int    `json:"wind_direction_deg" jsonschema:"wind direction in degrees (0=N, 90=E, 180=S, 270=W)"`
	WindSpeedUnit   string  `json:"wind_speed_unit" jsonschema:"wind speed unit symbol (e.g. km/h)"`
	WeatherCode     int     `json:"weather_code" jsonschema:"WMO weather interpretation code"`
	Weather         string  `json:"weather" jsonschema:"human-readable weather description"`
}

func validateUnits(units string) error {
	if units != "" && units != "metric" && units != "imperial" {
		return fmt.Errorf("units must be \"metric\" or \"imperial\", got %q", units)
	}
	return nil
}

func currentWeatherHandler(client *Client) mcp.ToolHandlerFor[CurrentInput, CurrentOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in CurrentInput) (*mcp.CallToolResult, CurrentOutput, error) {
		if err := validateUnits(in.Units); err != nil {
			return nil, CurrentOutput{}, err
		}
		loc, err := client.ResolveLocation(ctx, in.Location)
		if err != nil {
			return nil, CurrentOutput{}, err
		}
		cond, err := client.CurrentWeather(ctx, loc, in.Units)
		if err != nil {
			return nil, CurrentOutput{}, err
		}
		return nil, CurrentOutput{
			Location:         cond.Location.Name,
			Country:          cond.Location.Country,
			Time:             cond.Time,
			Temperature:      cond.Temperature,
			FeelsLike:        cond.FeelsLike,
			TemperatureUnit:  cond.TemperatureUnit,
			HumidityPercent:  cond.Humidity,
			PrecipitationMm:  cond.Precipitation,
			WindSpeed:        cond.WindSpeed,
			WindDirectionDeg: cond.WindDirection,
			WindSpeedUnit:    cond.WindSpeedUnit,
			WeatherCode:      cond.WeatherCode,
			Weather:          cond.WeatherDesc,
		}, nil
	}
}
