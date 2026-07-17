package handlers

// Dependencies aggregates injectable handler collaborators.
type Dependencies struct {
	Logger       Logger
	Authorizer   Authorizer
	Points       Points
	Search       Search
	History      History
	Responder    Responder
	HistoryLimit int
}

// withDefaults applies safe handler defaults.
func (d Dependencies) withDefaults() Dependencies {
	if d.Logger == nil {
		d.Logger = nopLogger{}
	}
	if d.HistoryLimit <= 0 {
		d.HistoryLimit = 10
	}
	return d
}

type nopLogger struct{}

func (nopLogger) Debugf(format string, args ...any) {}
func (nopLogger) Infof(format string, args ...any)  {}
func (nopLogger) Warnf(format string, args ...any)  {}
func (nopLogger) Errorf(format string, args ...any) {}
