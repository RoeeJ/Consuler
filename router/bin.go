package main

import (
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
	r := New(9090, m)
	r.Start()
}
