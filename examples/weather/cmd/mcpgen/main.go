package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ktr0731/go-mcp/codegen"
)

func main() {
	// Create output directory
	weatherDir := "."
	if err := os.MkdirAll(weatherDir, 0755); err != nil {
		log.Fatalf("Failed to create weather directory: %v", err)
	}

	// Create output file
	f, err := os.Create(filepath.Join(weatherDir, "mcp.gen.go"))
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Server definition
	def := &codegen.ServerDefinition{
		Capabilities: codegen.ServerCapabilities{
			Prompts: &codegen.PromptCapability{},
			Resources: &codegen.ResourceCapability{
				Subscribe:   true,
				ListChanged: true,
			},
			Tools:       &codegen.ToolCapability{},
			Logging:     &codegen.LoggingCapability{},
			Completions: &codegen.CompletionsCapability{},
		},
		Implementation: codegen.Implementation{
			Name:    "Weather Forecast MCP Server",
			Version: "1.0.0",
		},
		// Prompt definitions
		Prompts: []codegen.Prompt{
			{
				Name:        "weather_report",
				Description: "Generate a weather report based on weather data",
				Arguments: []codegen.PromptArgument{
					{Name: "city", Description: "City name", Required: true},
					{Name: "language", Description: "Report language (e.g. 'en', 'ja')", Required: false},
				},
			},
			{
				Name:        "weather_alert",
				Description: "Generate a weather alert message",
				Arguments: []codegen.PromptArgument{
					{Name: "alert_type", Description: "Type of alert (e.g. 'rain', 'snow', 'heat')", Required: true},
					{Name: "severity", Description: "Alert severity (1-5)", Required: true},
				},
			},
		},
		// Tool definitions
		Tools: []codegen.Tool{
			{
				Name:        "convert_temperature",
				Description: "Convert temperature between Celsius and Fahrenheit",
				InputSchema: struct {
					Temperature float64 `json:"temperature" jsonschema:"description=Temperature value to convert"`
					FromUnit    string  `json:"from_unit" jsonschema:"description=Source temperature unit,enum=celsius,enum=fahrenheit"`
					ToUnit      string  `json:"to_unit" jsonschema:"description=Target temperature unit,enum=celsius,enum=fahrenheit"`
				}{},
			},
			{
				Name:        "calculate_humidity_index",
				Description: "Calculate humidity index based on temperature and humidity",
				InputSchema: struct {
					Temperature float64 `json:"temperature" jsonschema:"description=Temperature in Celsius"`
					Humidity    float64 `json:"humidity" jsonschema:"description=Relative humidity percentage (0-100)"`
				}{},
			},
		},
		// Resource template definitions
		ResourceTemplates: []codegen.ResourceTemplate{
			{
				URITemplate: "weather://forecast/{city}",
				Name:        "City Weather Forecast",
				Description: "Weather forecast for a specific city",
				MimeType:    "application/json",
			},
			{
				URITemplate: "weather://historical/{city}/{date}",
				Name:        "Historical Weather Data",
				Description: "Historical weather data for a specific city and date",
				MimeType:    "application/json",
			},
		},
	}

	// Code generation
	if err := codegen.Generate(f, def, "weather"); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
