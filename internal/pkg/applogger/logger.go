package applogger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoggerCfg struct {
	FormatTimestamp string
	FormatLevel     string
	Loglvl          string
}

func init() {
	output := zerolog.ConsoleWriter{
		Out: os.Stdout,
		FormatTimestamp: func(any) string {
			return time.DateTime
		},
		FormatLevel: func(i any) string {
			return strings.ToUpper(fmt.Sprintf("%-6s", i))
		},
	}

	log.Logger = zerolog.New(output).With().
		Timestamp().CallerWithSkipFrameCount(2).Logger()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func Configure(cfg LoggerCfg) {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: cfg.FormatTimestamp,
		FormatLevel: func(i any) string {
			return strings.ToUpper(fmt.Sprintf(cfg.FormatLevel, i))
		},
	}

	log.Logger = zerolog.New(output).With().
		Timestamp().CallerWithSkipFrameCount(2).Logger()

	lvl, err := zerolog.ParseLevel(cfg.Loglvl)
	if err != nil {
		log.Warn().Err(fmt.Errorf("parseLevel: %w", err)).Msg("applogger.Configure")
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
}
