package spotify

import (
	"context"
	"fmt"
	"log/slog"

	librespot "github.com/devgianlu/go-librespot"
)

type SlogAdapter struct {
	log *slog.Logger
}

func (a SlogAdapter) Tracef(format string, args ...interface{}) {
	if a.log.Enabled(context.Background(), slog.LevelDebug) {
		a.log.Debug(fmt.Sprintf(format, args...))
	}
}

func (a SlogAdapter) Debugf(format string, args ...interface{}) {
	if a.log.Enabled(context.Background(), slog.LevelDebug) {
		a.log.Debug(fmt.Sprintf(format, args...))
	}
}

func (a SlogAdapter) Infof(format string, args ...interface{}) {
	a.log.Info(fmt.Sprintf(format, args...))
}

func (a SlogAdapter) Warnf(format string, args ...interface{}) {
	a.log.Warn(fmt.Sprintf(format, args...))
}

func (a SlogAdapter) Errorf(format string, args ...interface{}) {
	a.log.Error(fmt.Sprintf(format, args...))
}

func (a SlogAdapter) Trace(args ...interface{}) {
	if a.log.Enabled(context.Background(), slog.LevelDebug) {
		a.log.Debug(fmt.Sprint(args...))
	}
}

func (a SlogAdapter) Debug(args ...interface{}) {
	if a.log.Enabled(context.Background(), slog.LevelDebug) {
		a.log.Debug(fmt.Sprint(args...))
	}
}

func (a SlogAdapter) Info(args ...interface{}) {
	a.log.Info(fmt.Sprint(args...))
}

func (a SlogAdapter) Warn(args ...interface{}) {
	a.log.Warn(fmt.Sprint(args...))
}

func (a SlogAdapter) Error(args ...interface{}) {
	a.log.Error(fmt.Sprint(args...))
}

func (a SlogAdapter) WithField(key string, value interface{}) librespot.Logger {
	return SlogAdapter{a.log.With(key, value)}
}

func (a SlogAdapter) WithError(err error) librespot.Logger {
	return SlogAdapter{a.log.With("error", err)}
}
