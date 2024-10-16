package log

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	*zerolog.Logger
}

var defaultWriter = newConsoleWriter()

func newConsoleWriter() io.Writer {
	writer := zerolog.NewConsoleWriter()
	writer.TimeFormat = time.RFC3339
	return writer
}

func NewDefaultLogger() *Logger {
	return NewLogger(nil)
}

func GetDefaultWriter() io.Writer {
	return defaultWriter
}

func NewLogger(writer io.Writer) *Logger {
	if writer == nil {
		writer = defaultWriter
	}
	zl := zerolog.New(writer).With().Timestamp().Logger()

	return &Logger{&zl}
}

func (l *Logger) Debugf(format string, vaargs ...interface{}) {
	l.Debug().Msgf(format, vaargs...)
}

func (l *Logger) Infof(format string, vaargs ...interface{}) {
	l.Info().Msgf(format, vaargs...)
}

func (l *Logger) Warnf(format string, vaargs ...interface{}) {
	l.Warn().Msgf(format, vaargs...)
}

func (l *Logger) Errorf(format string, vaargs ...interface{}) {
	l.Error().Msgf(format, vaargs...)
}

func (l *Logger) Fatalf(format string, vaargs ...interface{}) {
	l.Fatal().Msgf(format, vaargs...)
}

func (l *Logger) Panicf(format string, vaargs ...interface{}) {
	l.Panic().Msgf(format, vaargs...)
}

func (l *Logger) Tracef(format string, vaargs ...interface{}) {
	l.Trace().Msgf(format, vaargs...)
}
