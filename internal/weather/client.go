// Package weather is an HTTP client for the Open-Meteo API (free, no API key).
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	geocodingBase = "https://geocoding-api.open-meteo.com/v1"
	forecastBase  = "https://api.open-meteo.com/v1"
)

// Client calls Open-Meteo.
type Client struct {
	http *http.Client
}

// NewClient returns a Client with the given per-request timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{http: &http.Client{Timeout: timeout}}
}

// Location is a resolved geographic point.
type Location struct {
	Name      string
	Latitude  float64
	Longitude float64
	Country   string
	Timezone  string
}

// ResolveLocation parses "lat,lon" coordinates or geocodes a city name.
func (c *Client) ResolveLocation(ctx context.Context, input string) (Location, error) {
	if parts := strings.SplitN(input, ",", 2); len(parts) == 2 {
		lat, errLat := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lon, errLon := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errLat == nil && errLon == nil {
			return Location{Name: input, Latitude: lat, Longitude: lon}, nil
		}
	}
	return c.geocode(ctx, input)
}

func (c *Client) geocode(ctx context.Context, name string) (Location, error) {
	u := geocodingBase + "/search?" + url.Values{
		"name": {name}, "count": {"1"}, "language": {"en"}, "format": {"json"},
	}.Encode()

	var resp struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   string  `json:"country"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}
	if err := c.get(ctx, u, &resp); err != nil {
		return Location{}, fmt.Errorf("geocode %q: %w", name, err)
	}
	if len(resp.Results) == 0 {
		return Location{}, fmt.Errorf("location not found: %q", name)
	}
	r := resp.Results[0]
	return Location{Name: r.Name, Latitude: r.Latitude, Longitude: r.Longitude, Country: r.Country, Timezone: r.Timezone}, nil
}

// CurrentConditions holds the live weather snapshot for a location.
type CurrentConditions struct {
	Location        Location
	Time            string
	Temperature     float64
	FeelsLike       float64
	Humidity        int
	Precipitation   float64
	WindSpeed       float64
	WindDirection   int
	WeatherCode     int
	WeatherDesc     string
	TemperatureUnit string
	WindSpeedUnit   string
}

// CurrentWeather fetches the current weather for loc. units is "metric" or "imperial".
func (c *Client) CurrentWeather(ctx context.Context, loc Location, units string) (CurrentConditions, error) {
	tempUnit, windUnit := resolveUnits(units)

	u := forecastBase + "/forecast?" + url.Values{
		"latitude":         {fmtCoord(loc.Latitude)},
		"longitude":        {fmtCoord(loc.Longitude)},
		"current":          {"temperature_2m,apparent_temperature,relative_humidity_2m,precipitation,wind_speed_10m,wind_direction_10m,weather_code"},
		"temperature_unit": {tempUnit},
		"wind_speed_unit":  {windUnit},
		"forecast_days":    {"1"},
	}.Encode()

	var resp struct {
		Current struct {
			Time                string  `json:"time"`
			Temperature2m       float64 `json:"temperature_2m"`
			ApparentTemperature float64 `json:"apparent_temperature"`
			RelativeHumidity2m  int     `json:"relative_humidity_2m"`
			Precipitation       float64 `json:"precipitation"`
			WindSpeed10m        float64 `json:"wind_speed_10m"`
			WindDirection10m    int     `json:"wind_direction_10m"`
			WeatherCode         int     `json:"weather_code"`
		} `json:"current"`
		CurrentUnits struct {
			Temperature2m string `json:"temperature_2m"`
			WindSpeed10m  string `json:"wind_speed_10m"`
		} `json:"current_units"`
	}
	if err := c.get(ctx, u, &resp); err != nil {
		return CurrentConditions{}, fmt.Errorf("current weather for %s: %w", loc.Name, err)
	}

	cur := resp.Current
	return CurrentConditions{
		Location:        loc,
		Time:            cur.Time,
		Temperature:     cur.Temperature2m,
		FeelsLike:       cur.ApparentTemperature,
		Humidity:        cur.RelativeHumidity2m,
		Precipitation:   cur.Precipitation,
		WindSpeed:       cur.WindSpeed10m,
		WindDirection:   cur.WindDirection10m,
		WeatherCode:     cur.WeatherCode,
		WeatherDesc:     DescribeWeatherCode(cur.WeatherCode),
		TemperatureUnit: resp.CurrentUnits.Temperature2m,
		WindSpeedUnit:   resp.CurrentUnits.WindSpeed10m,
	}, nil
}

// DayForecast holds the forecast for a single day.
type DayForecast struct {
	Date          string
	TempMax       float64
	TempMin       float64
	Precipitation float64
	WindSpeedMax  float64
	WeatherCode   int
	WeatherDesc   string
}

// ForecastResult is the multi-day forecast for a location.
type ForecastResult struct {
	Location        Location
	Days            []DayForecast
	TemperatureUnit string
	WindSpeedUnit   string
	PrecipUnit      string
}

// ForecastWeather fetches an n-day forecast for loc. days is clamped to [1, 16].
func (c *Client) ForecastWeather(ctx context.Context, loc Location, days int, units string) (ForecastResult, error) {
	if days <= 0 {
		days = 7
	}
	if days > 16 {
		days = 16
	}

	tempUnit, windUnit := resolveUnits(units)

	u := forecastBase + "/forecast?" + url.Values{
		"latitude":         {fmtCoord(loc.Latitude)},
		"longitude":        {fmtCoord(loc.Longitude)},
		"daily":            {"temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max,weather_code"},
		"temperature_unit": {tempUnit},
		"wind_speed_unit":  {windUnit},
		"forecast_days":    {strconv.Itoa(days)},
	}.Encode()

	var resp struct {
		Daily struct {
			Time             []string  `json:"time"`
			TempMax          []float64 `json:"temperature_2m_max"`
			TempMin          []float64 `json:"temperature_2m_min"`
			PrecipitationSum []float64 `json:"precipitation_sum"`
			WindSpeedMax     []float64 `json:"wind_speed_10m_max"`
			WeatherCode      []int     `json:"weather_code"`
		} `json:"daily"`
		DailyUnits struct {
			Temperature2mMax string `json:"temperature_2m_max"`
			WindSpeed10mMax  string `json:"wind_speed_10m_max"`
			PrecipitationSum string `json:"precipitation_sum"`
		} `json:"daily_units"`
	}
	if err := c.get(ctx, u, &resp); err != nil {
		return ForecastResult{}, fmt.Errorf("forecast for %s: %w", loc.Name, err)
	}

	d := resp.Daily
	dayForecasts := make([]DayForecast, len(d.Time))
	for i := range len(d.Time) {
		code := safeSlice(d.WeatherCode, i)
		dayForecasts[i] = DayForecast{
			Date:          safeSlice(d.Time, i),
			TempMax:       safeSlice(d.TempMax, i),
			TempMin:       safeSlice(d.TempMin, i),
			Precipitation: safeSlice(d.PrecipitationSum, i),
			WindSpeedMax:  safeSlice(d.WindSpeedMax, i),
			WeatherCode:   code,
			WeatherDesc:   DescribeWeatherCode(code),
		}
	}

	return ForecastResult{
		Location:        loc,
		Days:            dayForecasts,
		TemperatureUnit: resp.DailyUnits.Temperature2mMax,
		WindSpeedUnit:   resp.DailyUnits.WindSpeed10mMax,
		PrecipUnit:      resp.DailyUnits.PrecipitationSum,
	}, nil
}

func (c *Client) get(ctx context.Context, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from Open-Meteo", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func fmtCoord(f float64) string { return strconv.FormatFloat(f, 'f', 4, 64) }

func resolveUnits(units string) (tempUnit, windUnit string) {
	if units == "imperial" {
		return "fahrenheit", "mph"
	}
	return "celsius", "kmh"
}
func safeSlice[T any](ss []T, i int) (zero T) {
	if i < len(ss) {
		return ss[i]
	}
	return
}

