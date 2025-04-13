package main

import (
	"context"
	"log"

	"github.com/ktr0731/go-mcp/examples/weather"
)

func main() {
	if err := weather.Start(context.Background()); err != nil {
		log.Fatalf("failed to start weather server: %v", err)
	}
}
