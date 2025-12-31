package root

import "github.com/regenrek/peakypanes/internal/cli/spec"

// Handler executes a command.
type Handler func(ctx CommandContext) error

// Registry maps command IDs to handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds a handler for a command ID.
func (r *Registry) Register(id string, handler Handler) {
	if r == nil || id == "" || handler == nil {
		return
	}
	r.handlers[id] = handler
}

// HandlerFor returns a handler for the command ID.
func (r *Registry) HandlerFor(id string) (Handler, bool) {
	if r == nil {
		return nil, false
	}
	h, ok := r.handlers[id]
	return h, ok
}

// EnsureHandlers verifies that all leaf commands have handlers.
func (r *Registry) EnsureHandlers(spec *spec.Spec) error {
	if r == nil || spec == nil {
		return nil
	}
	for _, cmd := range spec.AllCommands() {
		if len(cmd.Subcommands) > 0 {
			continue
		}
		if _, ok := r.handlers[cmd.ID]; !ok {
			return missingHandlerError(cmd.ID)
		}
	}
	return nil
}
