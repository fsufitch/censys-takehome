package logging

import (
	"io"
	"os"
	"time"

	"github.com/fsufitch/censys-takehome/config"
	"github.com/rs/zerolog"
)

type LogFunc func() *zerolog.Logger

func ProvideLogFunc(conf config.LoggingConfiguration) LogFunc {
	level := zerolog.InfoLevel
	if conf.Debug {
		level = zerolog.DebugLevel
	}

	var writer io.WriteCloser = os.Stderr
	if conf.Pretty {
		writer = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr; w.TimeFormat = time.RFC3339 })
	}

	zlog := zerolog.New(writer).Level(level).With().Timestamp().Logger()

	return func() *zerolog.Logger { return &zlog }
}
