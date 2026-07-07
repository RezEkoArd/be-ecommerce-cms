package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(env string) {
	if env == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func withFields(e *zerolog.Event, field map[string]any) *zerolog.Event {
	for k, v := range field {
		e = e.Interface(k, v)
	}
	return e
}

func Info(msg string) {
	log.Info().Msg(msg)
}

func Infof(msg string, fields map[string]any) {
	withFields(log.Info(), fields).Msg(msg)
}

func Warn(msg string) {
	log.Warn().Msg(msg)
}

func Debugf(msg string, fields map[string]any) {
	withFields(log.Debug(), fields).Msg(msg)
}

func Error(msg string, err error) {
	log.Error().Err(err).Msg(msg)
}

func Errorf(msg string, err error, fields map[string]any) {
	withFields(log.Error().Err(err), fields).Msg(msg)
}

func Fatal(msg string, err error) {
	log.Fatal().Err(err).Msg(msg)
}
