package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	out := zerolog.NewConsoleWriter()
	logger := zerolog.New(out).With().Timestamp().Caller().Logger()
	log.Logger = logger
}
