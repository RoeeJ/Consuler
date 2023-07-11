package main

import (
	"github.com/nats-io/nats.go/micro"
	"math/rand"
	"time"

	morpheus "github.com/roeej/morpheus/core"
	"github.com/roeej/morpheus/core/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	rand.Seed(time.Now().Unix())
	m, err := morpheus.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
		return
	}

	logging.InitLogger()
	routerSvc := morpheus.Service{
		Name:        "router",
		Description: "Morpheus Router",
		Handler: func(request micro.Request) {
			log.Info().Msg("got request")
			_ = request.RespondJSON(map[string]string{"hello": "world"})
		},
	}
	_, err = m.RegisterService(&routerSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register service")
		return
	}
	r := New(9090, m)
	r.Start()
}
