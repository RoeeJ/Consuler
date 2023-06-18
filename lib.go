package main

import (
	"errors"
	capi "github.com/hashicorp/consul/api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

type ConsulInstance struct {
	name     string
	server   *ConsulServer
	client   *capi.Client
	services map[string]*ConsulRegistration
}
type ConsulServer struct {
	Addr   string
	Port   int
	Scheme string
}
type ConsulRegistration struct {
	Name          string
	Addr          string
	Port          int
	Registrations []capi.AgentServiceRegistration
	Checks        []capi.AgentServiceCheck
}

var (
	capiInstance *capi.Client
)

func (*ConsulRegistration) Register() error {
	return nil
}

func (*ConsulRegistration) Deregister() error {
	return nil
}

func (*ConsulRegistration) Update() error {
	return nil
}

func (ci *ConsulInstance) NewService(name string, addr string, port int) (*ConsulRegistration, error) {
	if ci == nil {
		return nil, errors.New("consul not initialized")
	}
	if ci.services[name] != nil {
		return nil, errors.New("service already exists")
	} else if name == "" || addr == "" || port == 0 {
		return nil, errors.New("invalid service definition")
	}
	reg := ConsulRegistration{
		Name:          name,
		Addr:          addr,
		Port:          port,
		Registrations: nil,
		Checks:        nil,
	}
	ci.services[name] = &reg
	return &reg, nil
}

func (ci *ConsulInstance) UpdateService(svc *ConsulRegistration) error {
	if ci == nil {
		return errors.New("consul not initialized")
	}
	if svc == nil {
		return errors.New("invalid service definition")
	}
	return nil
}

func (cr *ConsulRegistration) AddCheck(check capi.AgentServiceCheck) {
	cr.Checks = append(cr.Checks, check)
}
func NewInstance(name string, addr string, port int, scheme string) (*ConsulInstance, error) {
	conf := capi.DefaultConfig()
	conf.Address = addr
	conf.Scheme = scheme
	client, err := capi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	return &ConsulInstance{
		name: name,
		server: &ConsulServer{
			Addr:   addr,
			Port:   port,
			Scheme: scheme,
		},
		client:   client,
		services: make(map[string]*ConsulRegistration),
	}, nil
}
func initializeLogger() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()
	log.Logger = logger
}
func main() {
	initializeLogger()
	log.Info().Msg("Starting Morpheus")
}
