# 🚀 go-mcp

<div align="center">
  <h3>⚡ A type-safe, intuitive Go SDK for MCP server development</h3>
</div>

<p align="center">
  <a href="#installation">📥 Installation</a> •
  <a href="#features">✨ Features</a> •
  <a href="#quick-start">🏁 Quick Start</a> •
  <a href="#documentation">📚 Documentation</a> •
  <a href="#examples">🔍 Examples</a> •
  <a href="#contributing">🤝 Contributing</a>
</p>

## 🤔 What is go-mcp?

**go-mcp** is a Golang SDK for building MCP (Multi-agent Conversational Protocol) servers with ease and confidence. It provides a type-safe, intuitive interface that makes server development a breeze - even for those coming from the frontend world.

## ✨ Features

💪 **Go-Powered** - Leverage the performance and concurrency benefits of Go for your MCP servers

🔒 **Type-Safe** - Code generation ensures your tools and prompt parameters are statically typed, catching errors at compile time rather than runtime

🧩 **Simple & Intuitive API** - Natural, idiomatic Go interface that feels familiar and lets you build servers quickly without a steep learning curve

🚀 **Latest Protocol Support** - Always up-to-date with the newest MCP protocol versions

🔌 **Developer-Friendly** - Designed with API ergonomics in mind, making it approachable

## 📥 Installation

```bash
go get github.com/ktr0731/go-mcp
```

## 🏁 Quick Start

Creating an MCP server with go-mcp is straightforward!

### Define MCP server
First, create an executable file for code generation. Running this code will automatically generate files.

```go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ktr0731/go-mcp/codegen"
)

func main() {
	// Create output directory
	temperatureDir := "."
	if err := os.MkdirAll(temperatureDir, 0755); err != nil {
		log.Fatalf("Failed to create temperature directory: %v", err)
	}

	// Create output file
	f, err := os.Create(filepath.Join(temperatureDir, "mcp.gen.go"))
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Server definition
	def := &codegen.ServerDefinition{
		Capabilities: codegen.ServerCapabilities{
			Tools:       &codegen.ToolCapability{},
			Logging:     &codegen.LoggingCapability{},
		},
		Implementation: codegen.Implementation{
			Name:    "Temperature MCP Server",
			Version: "1.0.0",
		},
		// Tool definitions. Define by Go structs.
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
		},
	}

	// Code generation
	if err := codegen.Generate(f, def, "temperature"); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
```

Next, let's implement the MCP server using the generated files 🚀

```go
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
```

And that's it!

- A tool-specific type `ToolConvertTemperatureRequest` is generated
- Constants are generated from `enum` values
- No need to worry about JSON-RPC2.0 or MCP server core implementation

## 📚 Documentation

For detailed documentation, visit [the official docs](https://modelcontextprotocol.io).

## 🔍 Examples

Check out complete examples in the [examples directory](https://github.com/ktr0731/go-mcp/examples).

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. ⚖️
