package main

import (
	"context"
	"fmt"
	"log"
	"math"

	mcp "github.com/ktr0731/go-mcp"
	"golang.org/x/exp/jsonrpc2"
)

type toolHandler struct{}

func (h *toolHandler) HandleToolConvertTemperature(ctx context.Context, req *ToolConvertTemperatureRequest) (*mcp.CallToolResult, error) {
	temperature := req.Temperature
	fromUnit := req.FromUnit
	toUnit := req.ToUnit

	var result float64
	switch {
	case fromUnit == ConvertTemperatureFromUnitTypeCelsius && toUnit == ConvertTemperatureToUnitTypeFahrenheit:
		// C to F: (C * 9/5) + 32
		result = (temperature * 9 / 5) + 32
	case fromUnit == ConvertTemperatureFromUnitTypeFahrenheit && toUnit == ConvertTemperatureToUnitTypeCelsius:
		// F to C: (F - 32) * 5/9
		result = (temperature - 32) * 5 / 9
	case fromUnit == ConvertTemperatureFromUnitTypeCelsius && toUnit == ConvertTemperatureToUnitTypeCelsius ||
		fromUnit == ConvertTemperatureFromUnitTypeFahrenheit && toUnit == ConvertTemperatureToUnitTypeFahrenheit:
		result = temperature
	default:
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

func main() {
	handler := NewHandler(&toolHandler{})

	ctx, listener, binder := mcp.NewStdioTransport(context.Background(), handler, nil)
	srv, err := jsonrpc2.Serve(ctx, listener, binder)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	srv.Wait()
}
