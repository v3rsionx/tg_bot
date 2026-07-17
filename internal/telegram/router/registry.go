package router

import (
	"fmt"
	"sync"

	"github.com/go-telegram/bot"
)

// Registry is a thread-safe command handler registry.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]bot.HandlerFunc
}

// NewRegistry constructs an empty command registry.
func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]bot.HandlerFunc)}
}

// Register associates command name with a handler.
func (r *Registry) Register(command string, handler bot.HandlerFunc) {
	if r == nil || handler == nil || command == "" {
		return
	}
	r.mu.Lock()
	r.commands[command] = handler
	r.mu.Unlock()
}

// Get returns a registered command handler.
func (r *Registry) Get(command string) (bot.HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.commands[command]
	return handler, ok
}

// Commands returns registered command names.
func (r *Registry) Commands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// CommandHandlerSet is the subset of handlers required by the router.
type CommandHandlerSet interface {
	// Callback returns the shared callback-query handler.
	Callback() bot.HandlerFunc
	// Default returns the fallback update handler.
	Default() bot.HandlerFunc
}

// Apply registers command, callback, and default handlers on the bot client.
func Apply(client *bot.Bot, registry *Registry, handlers CommandHandlerSet) error {
	if client == nil {
		return fmt.Errorf("router: bot client is nil")
	}
	if registry == nil {
		return fmt.Errorf("router: registry is nil")
	}
	if handlers == nil {
		return fmt.Errorf("router: handlers is nil")
	}

	for _, name := range []string{"start", "help", "profile", "history", "admin"} {
		handler, ok := registry.Get(name)
		if !ok {
			return fmt.Errorf("router: missing command handler %q", name)
		}
		client.RegisterHandler(bot.HandlerTypeMessageText, "/"+name, bot.MatchTypePrefix, handler)
	}

	client.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, handlers.Callback())
	return nil
}
