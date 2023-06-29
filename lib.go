package morpheus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"strings"
	"time"
)

type Message struct {
	Timestamp int64
	From      string
	To        string
	Route     string
	Payload   interface{}
}
type Morpheus struct {
	client   *redis.Client
	context  context.Context
	Services []*Service
}

type Service struct {
	Id        string
	Name      string
	IpAddress string
	Port      int
	Routes    []string
	Chan      chan bool          `json:"-"` // used to signal that the service has been updated
	Handler   func(msg *Message) `json:"-"` // used to handle messages
}

const DefaultTTL = 10 * time.Second

func (s Service) Key() string {
	return fmt.Sprintf("morpheus:service:%s:%s", s.Name, s.Id)
}

func (s Service) getBaseKey() string {
	return fmt.Sprintf("morpheus:service:%s", s.Name)
}

func (m *Morpheus) Connect() error {
	host, ok := os.LookupEnv("REDIS_HOST")
	if !ok {
		host = "localhost:6379"
	}

	clientName := uuid.New().String()

	username := ""
	password := ""
	if un, ok := os.LookupEnv("REDIS_USERNAME"); ok {
		if pw, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
			username = un
			password = pw
		}
	}

	m.client = redis.NewClient(&redis.Options{
		Addr:       host,
		ClientName: clientName,
		Username:   username,
		Password:   password,
		DB:         0,
	})
	if m.context == nil {
		m.context = context.Background()
	}
	return nil
}
func (m *Morpheus) RegisterService(name string, port int, routes []string, handler func(msg *Message)) (*Service, error) {
	if m.serviceExists(name) {
		return nil, fmt.Errorf("service already exists")
	}
	svc := &Service{
		Id:        uuid.New().String(),
		Name:      name,
		IpAddress: m.getIpAddress(),
		Port:      port,
		Routes:    routes,
		Chan:      make(chan bool),
		Handler:   handler,
	}
	m.Services = append(m.Services, svc)
	m.UpdateService(svc)
	channels := []string{svc.Key(), svc.getBaseKey()}
	go func() {
		sub := m.client.Subscribe(m.context, channels...)
		log.Trace().Strs("channels", channels).Msg("subscribed to channels")
		for {
			select {
			case <-svc.Chan:
				break
			case msg := <-sub.Channel():
				var message Message
				err := json.Unmarshal([]byte(msg.Payload), &message)
				if err != nil {
					log.Error().Err(err).Msg("failed to unmarshal message")
					continue
				}
				handler(&message)
			default:
				continue
			}
		}
	}()
	go func() {
		for range time.Tick(5 * time.Second) {
			select {
			case <-svc.Chan:
				break
			default:
				m.UpdateService(svc)
			}
		}
	}()
	return svc, nil
}
func (m *Morpheus) internalListServices() int {
	return len(m.Services)
}
func (m *Morpheus) ListServices() []Service {
	keys := m.client.Keys(m.context, "morpheus:service:*:presence")
	if keys.Err() != nil {
		log.Error().Err(keys.Err()).Msg("failed to list services")
		return make([]Service, 0)
	}
	if len(keys.Val()) > 0 {
		svcs := make([]Service, 0)
		for _, key := range keys.Val() {
			val := m.client.Get(m.context, key)
			if val.Err() != nil {
				log.Error().Err(val.Err()).Msg("failed to get service")
				continue
			}
			svcName := getServiceName(key)
			for _, svc := range m.Services {
				if svc.Name == svcName {
					svcs = append(svcs, *svc)
				}
			}
		}
		return svcs
	}
	return make([]Service, 0)
}

func (m *Morpheus) DeleteService(svc Service) {

}
func (m *Morpheus) serviceExists(id string) bool {
	for _, svc := range m.Services {
		if svc.Id == id {
			return true
		}
	}
	return false
}

func (m *Morpheus) UpdatePresence(svc *Service) {
	key := fmt.Sprintf("%s:presence", svc.Key())
	m.client.Set(m.context, key, svc.Id, DefaultTTL)
}

func (m *Morpheus) UpdateService(svc *Service) {
	m.UpdatePresence(svc)
	m.UpdateHealth(svc)
	m.UpdateRoutes(svc)
}

func (m *Morpheus) UpdateHealth(svc *Service) {
	key := fmt.Sprintf("%s:health", svc.Key())
	jSvc, err := json.Marshal(svc)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal service")
		return
	}
	m.client.Set(m.context, key, string(jSvc), 10*time.Second)
}

func (m *Morpheus) UpdateRoutes(svc *Service) {
	key := fmt.Sprintf("%s:routes", svc.Key())
	m.client.SAdd(m.context, key, svc.Routes)
	m.client.Expire(m.context, key, DefaultTTL)
}

func (m *Morpheus) getIpAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal().Err(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (m *Morpheus) FlushDB() {
	_, err := m.client.FlushDB(m.context).Result()
	if err != nil {
		log.Error().Err(err).Msg("failed to flush db")
	}
}

func (m *Morpheus) Message(channel string, message Message) {
	for _, svc := range m.Services {
		if svc.Key() == channel || svc.getBaseKey() == channel {
			svc.Handler(&message)
			return
		}
	}
	jMsg, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal message")
		return
	}
	m.client.Publish(m.context, channel, string(jMsg))
}

func getBaseKey(key string) string {
	parts := strings.Split(key, ":")
	return strings.Join(parts[:3], ":")
}
func getServiceName(key string) string {
	parts := strings.Split(key, ":")
	return parts[2]
}

func Init() Morpheus {
	m := Morpheus{}
	err := m.Connect()
	if err != nil {
		panic(err)
	}
	return m
}

func InitLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	out := zerolog.NewConsoleWriter()
	logger := zerolog.New(out).With().Timestamp().Caller().Logger()
	log.Logger = logger
}
