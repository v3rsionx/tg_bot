// Package telegram implements the Telegram Bot transport layer.
//
// It wires go-telegram/bot with routing, middleware, and handlers while
// depending only on injectable interfaces for authorization, rate limiting,
// points, search, and history. No storage or search business logic lives here.
package telegram
