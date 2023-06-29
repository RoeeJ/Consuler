package main

import (
	"fmt"
	"github.com/roeej/morpheus"
	"github.com/rs/zerolog/log"
	"time"
)

func main() {
	morpheus.InitLogger()
	m := morpheus.Init()
	err := m.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
		return
	}
	_, err = m.RegisterService("test", 0, []string{
		"/test",
	}, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Sleeping for 30 seconds")
	time.Sleep(30 * time.Second)
}
