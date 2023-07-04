package morpheus

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
)

type Services map[string]map[string]*Service

func (s Services) Add(svc *Service) {
	if s[svc.Name] == nil {
		s[svc.Name] = make(map[string]*Service)
	}
	s[svc.Name][svc.Id] = svc
}

func (s Services) Remove(svc Service) {
	close(svc.LivenessChannel)
	delete(s[svc.Name], svc.Id)
} // map[service_name]map[service_id]*Service
type Service struct {
	Id              string    `json:"id,omitempty"`
	Name            string    `json:"name,omitempty"`
	IpAddress       string    `json:"ip_address,omitempty"`
	Port            int       `json:"port,omitempty"`
	Routes          Routes    `json:"routes,omitempty"`
	LivenessChannel chan bool `json:"-"`
}
type Routes []Route

func (s Routes) Len() int {
	return len(s)
}

func (s Routes) Match(path string) bool {
	for _, route := range s {
		if path == route.Route {
			log.Info().Str("path", path).Str("route", route.Route).Msg("matched route")
			return true
		}
	}
	return false
}

type Route struct {
	Route   string
	Handler MessageHandler `json:"-"` // used to handle messages
}

func (s Service) Key() string {
	return fmt.Sprintf("morpheus:service:%s:%s", s.Name, s.Id)
}

func (s Service) GetBaseKey() string {
	return fmt.Sprintf("morpheus:service:%s", s.Name)
}

func (s Service) Match(path string) bool {
	for _, route := range s.Routes {
		if strings.HasPrefix(path, route.Route) {
			return true
		}
	}
	return false
}
