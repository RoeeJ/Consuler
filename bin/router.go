package main

import (
	"github.com/roeej/morpheus"
	"github.com/roeej/morpheus/logging"
	"github.com/roeej/morpheus/router"
	"github.com/rs/zerolog/log"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	m, err := morpheus.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
		return
	}

	logging.InitLogger()
	r := router.NewRouter(8080, m)
	r.Start()
}
