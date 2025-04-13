package codegen_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ktr0731/go-mcp/codegen"
)

var update = flag.Bool("update", false, "update golden files")

func TestGenerate(t *testing.T) {
	t.Parallel()
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

	var buf bytes.Buffer
	if err := codegen.Generate(&buf, def, "weather"); err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	goldenDir := filepath.Join("testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("failed to create golden directory: %v", err)
	}

	goldenFile := filepath.Join(goldenDir, "weather_server.go.golden")
	if *update {
		if err := os.WriteFile(goldenFile, buf.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if string(expected) != buf.String() {
		t.Errorf("generated code does not match golden file")
	}
}
