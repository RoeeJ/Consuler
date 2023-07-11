package main

import (
	morpheus "github.com/roeej/morpheus/core"
	"github.com/roeej/morpheus/core/logging"
	"github.com/rs/zerolog/log"
	"testing"
)

// morpehus instance
var mi *morpheus.Morpheus

func TestMain(m *testing.M) {
	Setup()
	m.Run()
	Teardown()
}
func Setup() {
	logging.InitLogger()
	m, err := morpheus.Init(morpheus.Options{Mock: true})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to nats")
	}
	mi = m
}
func Teardown() {
}
