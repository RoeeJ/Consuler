package morpheus

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/rs/zerolog/log"
)

type Services map[string]*Service

func (s Services) Add(svc *Service) {
	s[svc.Name] = svc
}

func (s Services) Remove(svc Service) {
	err := svc.MicroService.Stop()
	if err != nil {
		log.Error().Err(err).Msg("failed to stop service")
		return
	}
	delete(s, svc.Name)
} // map[service_name]map[service_id]*Service
type Service struct {
	Name          string               `json:"name,omitempty"`
	Subscriptions []*nats.Subscription `json:"-"`
	Description   string               `json:"description,omitempty"`
	Handler       micro.HandlerFunc    `json:"-"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
	MicroService  micro.Service        `json:"-"`
}

func (s Service) Key() string {
	return fmt.Sprintf("morpheus.service.%s", s.Name)
}
