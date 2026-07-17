// Command bot is the Telegram bot application entry point.
//
// It wires configuration, logging, metrics, cache, SQLite, LMDB, search,
// business services, and Telegram transport, then runs until SIGINT/SIGTERM.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := buildApp(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bot startup failed: %v\n", err)
		os.Exit(1)
	}
	defer application.shutdown(context.Background())

	if err := application.run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "bot exited with error: %v\n", err)
		os.Exit(1)
	}
}
