package main

import (
	"fmt"

	"github.com/v3rsionx/tg_bot/internal/importer"
	applogger "github.com/v3rsionx/tg_bot/internal/logger"
)

type printfLogger struct {
	log applogger.Logger
}

func newPrintfLogger(log applogger.Logger) *printfLogger {
	if log == nil {
		log = applogger.Nop()
	}
	return &printfLogger{log: log}
}

func (l *printfLogger) Debugf(format string, args ...any) {
	l.log.Debug(fmt.Sprintf(format, args...))
}

func (l *printfLogger) Infof(format string, args ...any) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *printfLogger) Warnf(format string, args ...any) {
	l.log.Warn(fmt.Sprintf(format, args...))
}

func (l *printfLogger) Errorf(format string, args ...any) {
	l.log.Error(fmt.Sprintf(format, args...))
}

var _ importer.Logger = (*printfLogger)(nil)
