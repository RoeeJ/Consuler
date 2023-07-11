package morpheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/madflojo/tasks"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"
)

const DefaultTTL = 5 * time.Second

const DefaultHBInterval = 2 * time.Second

type Morpheus struct {
	client    *nats.Conn
	context   context.Context
	Services  Services // map[service_name]map[service_id]*Service
	Scheduler *tasks.Scheduler
	Options   Options
}

type MessageHandler func(m *Morpheus, msg *Message)

func (m *Morpheus) Connect() error {
	host, ok := os.LookupEnv("NATS_HOST")
	if !ok {
		host = "nats://127.0.0.1:4222"
	}

	clientName := randomId()

	username := ""
	password := ""
	if un, ok := os.LookupEnv("NATS_USERNAME"); ok {
		if pw, ok := os.LookupEnv("NATS_PASSWORD"); ok {
			username = un
			password = pw
		}
	}
	conn, err := nats.Connect(host, nats.Name(clientName), nats.UserInfo(username, password))
	if err != nil {
		return err
	}
	m.client = conn

	if m.context == nil {
		m.context = context.Background()
	}
	return nil
}

func (m *Morpheus) RegisterService(svc *Service) (*Service, error) {
	if svc == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}
	if m.serviceExists(svc.Name) {
		return nil, fmt.Errorf("service already exists")
	}
	service, err := micro.AddService(m.client, micro.Config{
		Name: svc.Name,
		Endpoint: &micro.EndpointConfig{
			Subject:  svc.Key(),
			Handler:  svc.Handler,
			Metadata: nil,
		},
		Version:     "0.0.1",
		Description: svc.Description,
		Metadata:    svc.Metadata,
	})
	if err != nil {
		return nil, err
	}
	svc.MicroService = service
	m.Services.Add(svc)
	return svc, nil
}

type ServiceInfo struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Version     string             `json:"version"`
	Metadata    *map[string]string `json:"metadata"`
	Description string             `json:"description"`
	Endpoints   []ServiceEndpoint  `json:"endpoints"`
}
type ServiceEndpoint struct {
	Name     string             `json:"name"`
	Subject  string             `json:"subject"`
	Metadata *map[string]string `json:"metadata"`
}

func (m *Morpheus) DeleteService(svc Service) error {
	if svc.MicroService != nil {
		return svc.MicroService.Stop()
	}
	return errors.New("service not found")
}

func (m *Morpheus) serviceExists(name string) bool {
	return m.Services[name] != nil
}

func (m *Morpheus) ListServices() *[]micro.Info {
	outch := make(chan *nats.Msg, 1000)
	resch := make(chan *[]micro.Info, 0)
	res := make([]micro.Info, 0)
	subjectPINGAll, _ := micro.ControlSubject(micro.InfoVerb, "", "")
	sub, err := m.client.Subscribe(m.client.NewRespInbox(), func(msg *nats.Msg) {
		outch <- msg
		var info micro.Info
		err := json.Unmarshal(msg.Data, &info)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal info")
			return
		}
		res = append(res, info)
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe")
		return nil
	}
	err = m.client.PublishRequest(subjectPINGAll, sub.Subject, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish request")
		return nil
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		resch <- &res
	}()
	return <-resch
}

func (m *Morpheus) RPC(name string, data []byte, headers nats.Header) (*nats.Msg, error) {
	subject := fmt.Sprintf("morpheus.service.%s", name)
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}
	return m.client.RequestMsg(msg, 100*time.Millisecond)
}

func getServiceName(key string) string {
	parts := strings.Split(key, ":")
	return parts[2]
}

type Options struct {
	Mock bool `json:"mock"` // if true, will not connect to redis
}

func Init(opts ...Options) (*Morpheus, error) {
	var options Options
	if len(opts) > 0 {
		options = opts[0]
	}
	m := Morpheus{
		Services:  make(Services),
		Scheduler: tasks.New(),
		Options:   options,
	}
	err := m.Connect()
	if err != nil {
		return nil, err
	}
	return &m, nil
}
