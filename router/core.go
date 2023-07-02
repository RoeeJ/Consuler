package router

import (
	"fmt"
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

func NewRouter(port int, m *morpheus.Morpheus) *Router {
	return &Router{
		Port:     port,
		Morpheus: m,
	}
}
func (r *Router) Start() {
	g := gin.Default()
	_ = g.SetTrustedProxies(nil)
	g.Any("/services", func(c *gin.Context) {
		c.JSONP(http.StatusOK, r.Morpheus.ListServices())
	})
	g.GET("/favicon.ico", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})
	g.Any("/svc/*svc", func(c *gin.Context) {
		reqpath := c.Param("svc")
		timeoutQ := c.DefaultQuery("timeout", "1")
		timeoutP, err := strconv.Atoi(timeoutQ)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		timeout := time.Duration(timeoutP) * time.Second
		svc, err := r.Morpheus.ResolveService(reqpath)
		if err != nil {
			_ = c.AbortWithError(http.StatusNotFound, err)
			return
		}
		msg := <-r.Morpheus.RPCWithTimeout(fmt.Sprintf("client:%s", c.ClientIP()), *svc, reqpath, nil, timeout)
		if msg == nil {
			c.AbortWithStatus(http.StatusBadGateway)
			return
		} else {
			c.JSON(http.StatusOK, msg)
		}
	})
	err := g.Run(fmt.Sprintf(":%d", r.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start router")
	}
}
