package logger

import "context"

type ctxKey struct{}

// ContextWithCorrelationID stores a correlation ID in ctx.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// CorrelationIDFromContext returns a correlation ID previously stored in ctx.
func CorrelationIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok && id != ""
}
