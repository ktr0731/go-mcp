# ⚡ go-mcp

<div align="center">
  <h3>A type‑safe, intuitive Go SDK for MCP server development</h3>
</div>

<p align="center">
  <a href="#-what-is-go-mcp">🤔 What is go‑mcp?</a> •
  <a href="#-features">✨ Features</a> •
  <a href="#-quick-start">🏁 Quick Start</a> •
  <a href="#-examples">🔍 Examples</a> •
  <a href="#-supported-features">✅ Supported Features</a> •
  <a href="#-contributing">🤝 Contributing</a>
</p>

---

## 🤔 What is go‑mcp?

**go‑mcp** is a Go SDK for building MCP (Model Context Protocol) servers with ease and confidence. It provides a type‑safe, intuitive interface that makes server development a breeze.

---

## ✨ Features

- 🔒 **Type‑Safe** – Code generation ensures your tools and prompt parameters are statically typed, so errors are caught at compile time instead of at runtime.
- 🧩 **Simple & Intuitive API** – A natural, idiomatic Go interface that lets you build servers quickly without a steep learning curve.
- 🔌 **Developer‑Friendly** – Designed with API ergonomics in mind, making it approachable.

---

## 🏁 Quick Start

Creating an MCP server with go‑mcp is straightforward!

### Directory structure

Below is an example directory structure for a temperature‑conversion MCP server:

```
.
├── cmd
│   ├── mcpgen
│   │   └── main.go
│   └── temperature
│       └── main.go
├── mcp.gen.go
└── temperature.go
```

### 1. Define the MCP server

First, create `cmd/mcpgen/main.go` for code generation. Running this file will automatically generate the necessary code.

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
    outDir := "."
    if err := os.MkdirAll(outDir, 0o755); err != nil {
        log.Fatalf("failed to create output directory: %v", err)
    }

    // Create output file
    f, err := os.Create(filepath.Join(outDir, "mcp.gen.go"))
    if err != nil {
        log.Fatalf("failed to create file: %v", err)
    }
    defer f.Close()

    // Server definition
    def := &codegen.ServerDefinition{
        Capabilities: codegen.ServerCapabilities{
            Tools:   &codegen.ToolCapability{},
            Logging: &codegen.LoggingCapability{},
        },
        Implementation: codegen.Implementation{
            Name:    "Temperature MCP Server",
            Version: "1.0.0",
        },
        // Tool definitions (declared with Go structs)
        Tools: []codegen.Tool{
            {
                Name:        "convert_temperature",
                Description: "Convert temperature between Celsius and Fahrenheit",
                InputSchema: struct {
                    Temperature float64 `json:"temperature" jsonschema:"description=Temperature value to convert"`
                    FromUnit    string  `json:"from_unit"  jsonschema:"description=Source temperature unit,enum=celsius,enum=fahrenheit"`
                    ToUnit      string  `json:"to_unit"    jsonschema:"description=Target temperature unit,enum=celsius,enum=fahrenheit"`
                }{},
            },
        },
    }

    // Generate code
    if err := codegen.Generate(f, def, "temperature"); err != nil {
        log.Fatalf("failed to generate code: %v", err)
    }
}
```

Generate the code:

```bash
go run ./cmd/mcpgen
```

### 2. Implement the MCP server

Next, implement the server logic in `cmd/temperature/main.go`:

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
        // °C → °F: (C × 9/5) + 32
        result = (temperature*9/5 + 32)
    case fromUnit == ConvertTemperatureFromUnitTypeFahrenheit && toUnit == ConvertTemperatureToUnitTypeCelsius:
        // °F → °C: (F − 32) × 5/9
        result = (temperature - 32) * 5 / 9
    case fromUnit == toUnit:
        result = temperature
    default:
        return nil, fmt.Errorf("unsupported conversion: %s to %s", fromUnit, toUnit)
    }

    // Round to two decimal places
    result = math.Round(result*100) / 100

    resultText := fmt.Sprintf("%.2f %s = %.2f %s", temperature, fromUnit, result, toUnit)

    return &mcp.CallToolResult{
        Content: []mcp.CallToolContent{
            mcp.TextContent{Text: resultText},
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

Run the server:

```bash
go run ./cmd/temperature
```

---

## 🔍 Examples

See complete examples in the [examples directory](https://github.com/ktr0731/go-mcp/tree/main/examples/weather) and the [API documentation](https://pkg.go.dev/github.com/ktr0731/go-mcp).

---

## ✅ Supported Features

- Ping
- Tools
- Prompts
- Prompts, Tools, Resources, Resource Templates
- Resource subscription
- Resource update notification
- Logging
- Completion
- Cancellation

🚧 **Under Development**

- Batching (JSON‑RPC 2.0)
- Streamable HTTP transport
- Progress notification

🚫 **Not Planned**

- Dynamic prompt and tool changes

  Go is not well‑suited for dynamic tool additions. Adding tools dynamically requires constructing tool definitions, JSON Schema, and handlers at runtime. While generated code remains type‑safe, dynamically added components do not, forcing heavy use of `any` and type assertions and harming interface consistency. We delegate these use cases to SDKs in languages better suited for dynamic changes, such as TypeScript.

  Most currently implemented MCP servers use static definitions only, and dynamic changes do not seem to be a primary use case yet.

---

## 🤝 Contributing

Contributions are welcome! Feel free to submit a pull request.

---

## 📄 License

This project is licensed under the MIT License – see the [LICENSE](LICENSE) file for details.