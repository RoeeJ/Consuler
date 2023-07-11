package main

import (
	"fmt"
	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	morpheus "github.com/roeej/morpheus/core"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

type Router struct {
	Port     int
	Morpheus *morpheus.Morpheus
}

func New(port int, m *morpheus.Morpheus) *Router {
	return &Router{
		Port:     port,
		Morpheus: m,
	}
}

func (r *Router) Start() {
	g := gin.Default()
	_ = g.SetTrustedProxies(nil)
	g.Use(favicon.New())
	g.GET("/health", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
	})
	g.GET("/services", func(c *gin.Context) {
		svcs := r.Morpheus.ListServices()
		c.JSONP(http.StatusOK, svcs)
	})
	g.GET("/rpc/*svc", HandleRPC(r))
	err := g.Run(fmt.Sprintf(":%d", r.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start router")
	}
}

func HandleRPC(r *Router) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		svcParam := strings.Split(strings.TrimLeft(c.Param("svc"), "/"), "/")
		svcName := svcParam[0]
		reqPath := []byte(strings.Join(svcParam[1:], "/"))
		resp, err := r.Morpheus.RPC(svcName, reqPath, nats.Header(c.Request.Header))
		if err != nil {
			_ = c.AbortWithError(http.StatusNotFound, err)
			return
		}
		for k, v := range resp.Header {
			c.Header(k, v[0])
		}
		c.String(http.StatusOK, string(resp.Data))
	}
	return fn
}
