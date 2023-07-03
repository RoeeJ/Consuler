package router

import (
	"fmt"
	"github.com/fufuok/favicon"
	"github.com/gin-gonic/gin"
	"github.com/roeej/morpheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"time"
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
		c.JSONP(http.StatusOK, r.Morpheus.ListServices())
	})
	g.GET("/rpc/*svc", HandleRPC(r))
	err := g.Run(fmt.Sprintf(":%d", r.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start router")
	}
}

func HandleRPC(r *Router) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		reqpath := c.Param("svc")
		timeoutQ := c.DefaultQuery("timeout", "1")
		timeoutP, err := strconv.ParseFloat(timeoutQ, 64)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		timeout := time.Duration(timeoutP * float64(time.Second))
		svc, err := r.Morpheus.ResolveService(reqpath)
		if err != nil {
			_ = c.AbortWithError(http.StatusNotFound, err)
			return
		}
		clientId := fmt.Sprintf("client:%s", c.ClientIP())
		headers := make(map[string]string)
		for k, v := range c.Request.Header {
			headers[k] = v[0]
		}
		msg := <-r.Morpheus.RPCWithTimeout(morpheus.FromServiceWithMeta(clientId, *svc, reqpath, nil, headers), timeout)
		if msg == nil {
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		} else {
			c.JSON(http.StatusOK, msg)
		}
	}
	return fn
}
