package main

import (
	"github.com/nats-io/nats.go/micro"
	morpheus "github.com/roeej/morpheus/core"
	"github.com/roeej/morpheus/core/logging"
	"github.com/roeej/morpheus/core/message"
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
	svc := morpheus.Service{
		Name:        "echo",
		Description: "Morpheus Echo Service",
		Handler: func(request micro.Request) {
			msg, err := message.FromNatsRequest(request)
			if err != nil {
				_ = request.Error(err.Error(), "failed to parse message", nil)
				return
			}
			_ = request.Respond(msg.JSON(), micro.WithHeaders(msg.Meta), micro.WithHeaders(msg.tracingHeaders()))
		},
	}
	_, err = m.RegisterService(&svc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register service")
		return
	}
	<-killch
}
