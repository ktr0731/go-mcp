package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ktr0731/go-mcp/codegen"
)

func main() {
	// Create output directory
	diceDir := "."
	if err := os.MkdirAll(diceDir, 0755); err != nil {
		log.Fatalf("Failed to create dice directory: %v", err)
	}

	// Create output file
	f, err := os.Create(filepath.Join(diceDir, "mcp.gen.go"))
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Server definition
	def := &codegen.ServerDefinition{
		Capabilities: codegen.ServerCapabilities{
			Tools:   &codegen.ToolCapability{},
			Logging: &codegen.LoggingCapability{},
		},
		Implementation: codegen.Implementation{
			Name:    "Dice MCP Server",
			Version: "1.0.0",
		},
		Tools: []codegen.Tool{
			{
				Name:        "roll_dice",
				Description: "Roll a dice",
				InputSchema: struct {
					Count int `json:"count" jsonschema:"description=Number of dice to roll"`
				}{},
			},
		},
	}

	// Code generation
	if err := codegen.Generate(f, def, "dice"); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
