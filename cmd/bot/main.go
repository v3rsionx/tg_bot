// Command bot is the Telegram bot application entry point.
package main

import (
	"log"

	"github.com/v3rsi/tgbot-versionx/internal/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		log.Fatal(err)
	}
}
