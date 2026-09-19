package weather

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ForecastInput struct {
	Location string `json:"location" jsonschema:"full official city name in Latin script (e.g. 'London', 'Moscow', 'Saint Petersburg', 'Berlin') or decimal coordinates as 'lat,lon' (e.g. '59.93,30.32'); always use the full official English name, not informal, abbreviated, or non-Latin forms"`
	Days     int    `json:"days,omitempty" jsonschema:"number of forecast days (1–16); use 7 for 'next week' queries; default is 7"`
	Units    string `json:"units,omitempty" jsonschema:"temperature units: 'metric' (°C, km/h) or 'imperial' (°F, mph); default is metric"`
}

type DayOutput struct {
	Date          string  `json:"date" jsonschema:"date in YYYY-MM-DD format"`
	TempMax       float64 `json:"temp_max" jsonschema:"maximum temperature of the day"`
	TempMin       float64 `json:"temp_min" jsonschema:"minimum temperature of the day"`
	Precipitation float64 `json:"precipitation" jsonschema:"total precipitation for the day"`
	WindSpeedMax  float64 `json:"wind_speed_max" jsonschema:"maximum wind speed of the day"`
	WeatherCode   int     `json:"weather_code" jsonschema:"WMO weather interpretation code"`
	Weather       string  `json:"weather" jsonschema:"human-readable weather description"`
}

type ForecastOutput struct {
	Location        string      `json:"location" jsonschema:"resolved location name"`
	Country         string      `json:"country,omitempty" jsonschema:"country name"`
	TemperatureUnit string      `json:"temperature_unit" jsonschema:"temperature unit symbol (e.g. °C)"`
	WindSpeedUnit   string      `json:"wind_speed_unit" jsonschema:"wind speed unit symbol (e.g. km/h)"`
	PrecipUnit      string      `json:"precipitation_unit" jsonschema:"precipitation unit symbol (e.g. mm)"`
	Days            []DayOutput `json:"days" jsonschema:"daily forecast entries"`
}

func forecastHandler(client *Client) mcp.ToolHandlerFor[ForecastInput, ForecastOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ForecastInput) (*mcp.CallToolResult, ForecastOutput, error) {
		if err := validateUnits(in.Units); err != nil {
			return nil, ForecastOutput{}, err
		}
		loc, err := client.ResolveLocation(ctx, in.Location)
		if err != nil {
			return nil, ForecastOutput{}, err
		}
		result, err := client.ForecastWeather(ctx, loc, in.Days, in.Units)
		if err != nil {
			return nil, ForecastOutput{}, err
		}

		days := make([]DayOutput, len(result.Days))
		for i, d := range result.Days {
			days[i] = DayOutput{
				Date:          d.Date,
				TempMax:       d.TempMax,
				TempMin:       d.TempMin,
				Precipitation: d.Precipitation,
				WindSpeedMax:  d.WindSpeedMax,
				WeatherCode:   d.WeatherCode,
				Weather:       d.WeatherDesc,
			}
		}

		return nil, ForecastOutput{
			Location:        result.Location.Name,
			Country:         result.Location.Country,
			TemperatureUnit: result.TemperatureUnit,
			WindSpeedUnit:   result.WindSpeedUnit,
			PrecipUnit:      result.PrecipUnit,
			Days:            days,
		}, nil
	}
}
