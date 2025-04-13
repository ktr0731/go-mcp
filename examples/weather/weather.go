package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	mcp "github.com/ktr0731/go-mcp"
	"golang.org/x/exp/jsonrpc2"
)

// CityWeather represents weather data for a city
type CityWeather struct {
	City        string    `json:"city"`
	Date        time.Time `json:"date"`
	Temperature float64   `json:"temperature"` // Celsius
	Humidity    float64   `json:"humidity"`    // Relative humidity (%)
	Condition   string    `json:"condition"`   // Weather condition (sunny, cloudy, rainy, etc.)
	WindSpeed   float64   `json:"windSpeed"`   // Wind speed (m/s)
}

type promptHandler struct {
	cities map[string]*CityWeather
}

var _ ServerPromptHandler = (*promptHandler)(nil)

func (h *promptHandler) HandlePromptWeatherReport(ctx context.Context, req *PromptWeatherReportRequest) (*mcp.GetPromptResult, error) {
	cityName := req.City
	city, ok := h.cities[cityName]
	if !ok {
		return nil, fmt.Errorf("city not found: %s", cityName)
	}

	// Set report language (default is English)
	language := "en"
	if req.Language != "" {
		language = req.Language
	}

	var reportText string
	if language == "ja" {
		reportText = fmt.Sprintf(
			"%sの天気レポートです。現在の気温は%.1f℃、湿度は%.1f%%、天候は%sで、風速は%.1fm/sです。",
			city.City, city.Temperature, city.Humidity, translateCondition(city.Condition, "ja"), city.WindSpeed,
		)
	} else {
		reportText = fmt.Sprintf(
			"Weather report for %s. Current temperature is %.1f°C, humidity is %.1f%%, weather condition is %s, and wind speed is %.1f m/s.",
			city.City, city.Temperature, city.Humidity, city.Condition, city.WindSpeed,
		)
	}

	return &mcp.GetPromptResult{
		Description: "Weather report for " + city.City,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Text: fmt.Sprintf("Please provide a weather report for %s", city.City),
				},
			},
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Text: reportText,
				},
			},
		},
	}, nil
}

func (h *promptHandler) HandlePromptWeatherAlert(ctx context.Context, req *PromptWeatherAlertRequest) (*mcp.GetPromptResult, error) {
	alertType := req.AlertType
	severity := req.Severity

	var alertText string
	switch alertType {
	case "rain":
		alertText = fmt.Sprintf("WEATHER ALERT: Heavy rain warning. Severity level: %s/5. Expect heavy rainfall and possible flooding in low-lying areas. Please take necessary precautions.", severity)
	case "snow":
		alertText = fmt.Sprintf("WEATHER ALERT: Snow warning. Severity level: %s/5. Expect heavy snowfall and difficult road conditions. Please avoid unnecessary travel.", severity)
	case "heat":
		alertText = fmt.Sprintf("WEATHER ALERT: Heat warning. Severity level: %s/5. Extremely high temperatures expected. Stay hydrated and avoid direct sun exposure.", severity)
	default:
		alertText = fmt.Sprintf("WEATHER ALERT: %s warning. Severity level: %s/5. Please stay informed about changing weather conditions.", alertType, severity)
	}

	return &mcp.GetPromptResult{
		Description: "Weather alert for " + alertType,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Text: fmt.Sprintf("Generate a weather alert for %s with severity %s", alertType, severity),
				},
			},
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Text: alertText,
				},
			},
		},
	}, nil
}

type toolHandler struct {
	cities map[string]*CityWeather
}

var _ ServerToolHandler = (*toolHandler)(nil)

func (h *toolHandler) HandleToolConvertTemperature(ctx context.Context, req *ToolConvertTemperatureRequest) (*mcp.CallToolResult, error) {
	temperature := req.Temperature
	fromUnit := req.FromUnit
	toUnit := req.ToUnit

	var result float64
	if fromUnit == ConvertTemperatureFromUnitTypeCelsius && toUnit == ConvertTemperatureToUnitTypeFahrenheit {
		// C to F: (C * 9/5) + 32
		result = (temperature * 9 / 5) + 32
	} else if fromUnit == ConvertTemperatureFromUnitTypeFahrenheit && toUnit == ConvertTemperatureToUnitTypeCelsius {
		// F to C: (F - 32) * 5/9
		result = (temperature - 32) * 5 / 9
	} else if fromUnit == ConvertTemperatureFromUnitTypeCelsius && toUnit == ConvertTemperatureToUnitTypeCelsius {
		result = temperature
	} else if fromUnit == ConvertTemperatureFromUnitTypeFahrenheit && toUnit == ConvertTemperatureToUnitTypeFahrenheit {
		result = temperature
	} else {
		return nil, fmt.Errorf("unsupported conversion: %s to %s", fromUnit, toUnit)
	}

	// Round to 2 decimal places
	result = math.Round(result*100) / 100

	resultText := fmt.Sprintf("%.2f %s = %.2f %s", temperature, fromUnit, result, toUnit)

	return &mcp.CallToolResult{
		Content: []mcp.CallToolContent{
			mcp.TextContent{
				Text: resultText,
			},
		},
	}, nil
}

func (h *toolHandler) HandleToolCalculateHumidityIndex(ctx context.Context, req *ToolCalculateHumidityIndexRequest) (*mcp.CallToolResult, error) {
	temperature := req.Temperature
	humidity := req.Humidity

	// Discomfort index simplified formula: 0.81 × temperature + 0.01 × humidity × (0.99 × temperature - 14.3) + 46.3
	index := 0.81*temperature + 0.01*humidity*(0.99*temperature-14.3) + 46.3
	// Round to 1 decimal place
	index = math.Round(index*10) / 10

	var comfort string
	switch {
	case index < 55:
		comfort = "Cold"
	case index < 60:
		comfort = "Slightly cool"
	case index < 65:
		comfort = "Comfortable"
	case index < 70:
		comfort = "Pleasant"
	case index < 75:
		comfort = "Slightly warm"
	case index < 80:
		comfort = "Warm"
	case index < 85:
		comfort = "Hot"
	default:
		comfort = "Very hot"
	}

	resultText := fmt.Sprintf("Temperature: %.1f°C, Humidity: %.1f%%\nComfort Index: %.1f (%s)", temperature, humidity, index, comfort)

	return &mcp.CallToolResult{
		Content: []mcp.CallToolContent{
			mcp.TextContent{
				Text: resultText,
			},
		},
	}, nil
}

type resourceHandler struct {
	cities map[string]*CityWeather
}

var _ mcp.ServerResourceHandler = (*resourceHandler)(nil)

func (h *resourceHandler) HandleResourcesList(ctx context.Context) (*mcp.ListResourcesResult, error) {
	resources := []mcp.Resource{}

	// Add resources for each city
	for id, city := range h.cities {
		resources = append(resources, mcp.Resource{
			URI:         fmt.Sprintf("weather://forecast/%s", id),
			Name:        fmt.Sprintf("%s Weather Forecast", city.City),
			Description: fmt.Sprintf("Current weather data for %s", city.City),
			MimeType:    "application/json",
		})
	}

	return &mcp.ListResourcesResult{
		Resources: resources,
	}, nil
}

func (h *resourceHandler) HandleResourcesRead(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.URI

	// Extract path from URI: weather://forecast/tokyo → tokyo
	path := uri[len("weather://forecast/"):]

	// Get weather data for the city
	city, ok := h.cities[path]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	// Convert to JSON
	weatherJSON, err := json.MarshalIndent(city, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal city data: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContent{
			mcp.TextResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     string(weatherJSON),
			},
		},
	}, nil
}

// translateCondition translates weather conditions to the specified language
func translateCondition(condition, language string) string {
	if language != "ja" {
		return condition
	}

	translations := map[string]string{
		"sunny":    "晴れ",
		"cloudy":   "曇り",
		"rainy":    "雨",
		"snowy":    "雪",
		"foggy":    "霧",
		"windy":    "強風",
		"stormy":   "嵐",
		"thunder":  "雷",
		"drizzle":  "小雨",
		"blizzard": "吹雪",
	}

	if translation, ok := translations[condition]; ok {
		return translation
	}
	return condition
}

type completionHandler struct {
	cities map[string]*CityWeather
}

var _ mcp.ServerCompletionHandler = (*completionHandler)(nil)

func (h *completionHandler) HandleComplete(ctx context.Context, req *mcp.CompleteRequestParams) (*mcp.CompleteResult, error) {
	logger := mcp.Logger(ctx, "completionHandler").With("ref", req.Ref, "argument", req.Argument)
	logger.Info("called complete")

	if req.Ref.Type == mcp.CompletionReferenceTypePrompt {
		switch req.Argument.Name {
		case "city":
			values := []string{}
			for id := range h.cities {
				if strings.Contains(strings.ToLower(id), req.Argument.Value) {
					values = append(values, id)
				}
			}
			return &mcp.CompleteResult{
				Values: values,
			}, nil
		case "language":
			return &mcp.CompleteResult{
				Values: []string{"en", "ja"},
			}, nil
		}
	} else if req.Ref.Type == mcp.CompletionReferenceTypeResource {
		switch req.Argument.Name {
		case "city":
			values := []string{}
			for id := range h.cities {
				if strings.Contains(strings.ToLower(id), req.Argument.Value) {
					values = append(values, id)
				}
			}
			return &mcp.CompleteResult{
				Values: values,
			}, nil
		}
	}
	return nil, fmt.Errorf("unsupported reference: %+v", req.Ref)
}

// Start launches the MCP server.
func Start(ctx context.Context) error {
	// Create sample data
	cities := make(map[string]*CityWeather)

	// Add sample data
	cities["tokyo"] = &CityWeather{
		City:        "Tokyo",
		Date:        time.Now(),
		Temperature: 22.5,
		Humidity:    65.0,
		Condition:   "sunny",
		WindSpeed:   3.2,
	}

	cities["new_york"] = &CityWeather{
		City:        "New York",
		Date:        time.Now(),
		Temperature: 18.2,
		Humidity:    70.0,
		Condition:   "cloudy",
		WindSpeed:   5.1,
	}

	cities["london"] = &CityWeather{
		City:        "London",
		Date:        time.Now(),
		Temperature: 15.8,
		Humidity:    75.0,
		Condition:   "rainy",
		WindSpeed:   4.0,
	}

	promptHandler := &promptHandler{cities: cities}
	toolHandler := &toolHandler{cities: cities}
	completionHandler := &completionHandler{cities: cities}
	resourceHandler := &resourceHandler{cities: cities}

	handler := NewHandler(promptHandler, resourceHandler, toolHandler, completionHandler)

	ctx, listener, binder := mcp.NewStdioTransport(ctx, handler, nil)
	srv, err := jsonrpc2.Serve(ctx, listener, binder)
	if err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return srv.Wait()
}
