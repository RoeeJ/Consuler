package main

import (
	"context"
	morpheus "github.com/roeej/morpheus/core"
	"github.com/roeej/morpheus/core/logging"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

func main() {
	killch := make(chan os.Signal, 1)
	signal.Notify(killch, os.Interrupt)
	logging.InitLogger()
	m, err := morpheus.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
		return
	}
	_, err = m.RegisterService("echo", 0, morpheus.Routes{
		morpheus.Route{
			Route:   "/",
			Handler: HandleMessage,
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register service")
		return
	}
	<-killch
}

func HandleMessage(m *morpheus.Morpheus, msg *morpheus.Message) {
	m.Respond(context.Background(), msg, msg)
}
