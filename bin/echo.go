package main

import (
	"github.com/roeej/morpheus"
	"github.com/roeej/morpheus/logging"
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
	service, err := m.RegisterService("echo", 0, morpheus.ServiceRoutes{
		morpheus.ServiceRoute{
			Route:   "/echo",
			Handler: HandleMessage,
		},
	}, HandleMessage)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register service")
		return
	}
	log.Debug().Interface("service", service).Msg("registered service")
	<-killch
}

func HandleMessage(m *morpheus.Morpheus, msg *morpheus.Message) {
	switch msg.Route {
	case "/echo":
		m.Respond(msg, msg)
	}
}
