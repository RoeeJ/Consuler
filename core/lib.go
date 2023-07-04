package morpheus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/madflojo/tasks"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

const DefaultTTL = 5 * time.Second

const DefaultHBInterval = 2 * time.Second

type Morpheus struct {
	client    *redis.Client
	context   context.Context
	Services  Services // map[service_name]map[service_id]*Service
	Scheduler *tasks.Scheduler
	Options   Options
}

type MessageHandler func(m *Morpheus, msg *Message)

func (m *Morpheus) Connect() error {
	host, ok := os.LookupEnv("REDIS_HOST")
	if !ok {
		host = "localhost:6379"
	}

	clientName := randomId()

	username := ""
	password := ""
	if un, ok := os.LookupEnv("REDIS_USERNAME"); ok {
		if pw, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
			username = un
			password = pw
		}
	}
	if m.Options.Mock {
		mr, err := miniredis.Run()
		if err != nil {
			return fmt.Errorf("failed to create miniredis instance: %s", err)
		}
		m.client = redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
	} else {
		m.client = redis.NewClient(&redis.Options{
			Addr:       host,
			ClientName: clientName,
			Username:   username,
			Password:   password,
			DB:         0,
		})
	}
	if m.context == nil {
		m.context = context.Background()
	}
	return nil
}

func (m *Morpheus) RegisterService(name string, port int, routes Routes) (*Service, error) {
	svcId := randomId()
	if m.serviceExists(name, svcId) {
		return nil, fmt.Errorf("service already exists")
	}
	svc := &Service{
		Id:        svcId,
		Name:      name,
		IpAddress: m.GetDefaultIP(),
		Port:      port,
		Routes:    routes,
	}
	m.Services.Add(svc)
	m.UpdateService(svc)
	channels := []string{svc.Key(), svc.GetBaseKey()}
	sub := m.client.Subscribe(m.context, channels...)
	go func() {
		log.Trace().Strs("channels", channels).Msg("subscribed to channels")
		rCh := sub.Channel()
		for {
			select {
			case _msg := <-rCh:
				var msg Message
				err := json.Unmarshal([]byte(_msg.Payload), &msg)
				if err != nil {
					log.Error().Err(err).Msg("failed to unmarshal message")
					continue
				}
				for _, route := range svc.Routes {
					if strings.HasPrefix(msg.Route, route.Route) {
						route.Handler(m, &msg)
					}
				}
			case <-svc.LivenessChannel:
				log.Warn().Str("id", svc.Id).Msg("service liveness channel closed")
				return
			}
		}
	}()
	err := m.Scheduler.AddWithID(svc.Id, &tasks.Task{
		Interval: DefaultHBInterval,
		TaskFunc: func() error {
			log.Trace().Str("id", svc.Id).Msg("updating service")
			m.UpdateService(svc)
			return nil
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to add task")
	}
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
			svc, err := m.FetchService(svcName, val.Val())
			if err != nil {
				log.Error().Err(err).Msg("failed to fetch service")
				continue
			}
			svcs = append(svcs, svc)
		}
		sort.SliceStable(svcs, func(i, j int) bool {
			return svcs[i].Id < svcs[j].Id
		})
		return svcs
	}
	return make([]Service, 0)
}

func (m *Morpheus) DeleteService(svc Service) {
	delete(m.Services, svc.Id)
}

func (m *Morpheus) serviceExists(name string, id string) bool {
	return m.Services[name][id] != nil
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
	m.client.Set(m.context, key, string(jSvc), DefaultTTL)
}

func (m *Morpheus) UpdateRoutes(svc *Service) {
	key := fmt.Sprintf("%s:routes", svc.Key())
	routes := make([]string, 0)
	for _, route := range svc.Routes {
		routes = append(routes, route.Route)
	}
	m.client.SAdd(m.context, key, routes)
	m.client.Expire(m.context, key, DefaultTTL)
}

func (m *Morpheus) GetDefaultIP() string {
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

//func (m *Morpheus) RPC(from string, service Service, route string, meta map[string]string, payload interface{}) *Message {
//	retch := make(chan *Message)
//	m.Message(m.context, service.Key(), from, service.Key(), route, meta, payload, nil, retch)
//	return <-retch
//}
//
//func (m *Morpheus) RPCWithTimeout(from string, service Service, route string, meta map[string]string, payload interface{}, timeout time.Duration) *Message {
//	retch := make(chan *Message, 1)
//	ctx, _ := context.WithTimeout(m.context, timeout)
//	m.Message(ctx, service.Key(), from, service.Key(), route, meta, payload, nil, retch)
//	select {
//	case msg := <-retch:
//		return msg
//	case <-ctx.Done():
//		return nil
//	}
//}

//	func (m *Morpheus) Respond(msg *Message, payload interface{}) {
//		m.RespondWithHeaders(msg, nil, payload)
//	}
//
//	func (m *Morpheus) RespondWithHeaders(msg *Message, headers map[string]string, payload interface{}) {
//		m.Message(m.context, msg.ResponseChannel, msg.To, msg.From, msg.Route, headers, payload, &msg.ResponseChannel, nil)
//	}
func (m *Morpheus) Send(ctx context.Context, msg Message) {
	r := m.client.Publish(ctx, msg.Channel, msg.Json())
	if r.Err() != nil {
		log.Error().Err(r.Err()).Msg("failed to send message")
	}
}

func (m *Morpheus) SendWithTimeout(msg Message, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(m.context, timeout)
	go func() {
		<-time.After(timeout)
		cancel()
	}()
	m.client.Publish(ctx, msg.Channel, msg)
}

func (m *Morpheus) RPC(msg Message) chan *Message {
	retch := make(chan *Message)
	go func() {
		retch <- <-m.ReceiveMessage(m.context, msg.ResponseChannel, time.Hour)
	}()
	m.Send(m.context, msg) //Message(m.context, msg.Channel, msg.From, msg.To, msg.Route, msg.Meta, msg.Payload, nil, retch)
	return retch
}

func (m *Morpheus) RPCWithTimeout(msg Message, timeout time.Duration) chan *Message {
	ctx, cancel := context.WithTimeout(m.context, timeout)
	retch := make(chan *Message)
	go func() {
		sub := m.client.Subscribe(ctx, msg.ResponseChannel)
		m.Send(ctx, msg)
		select {
		case <-ctx.Done():
			close(retch)
		case receiveMessage := <-sub.Channel():
			if receiveMessage == nil {
				close(retch)
			} else {
				response := FromRedisMessage(receiveMessage)
				retch <- &response
			}
		}
		cancel()
	}()
	return retch
}

func (m *Morpheus) ReceiveMessage(ctx context.Context, channel string, timeout time.Duration) chan *Message {
	retch := make(chan *Message)
	go func() {
		sub := m.client.Subscribe(ctx, channel)
		for {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				log.Error().Msg("failed to receive message")
				cancel()
				return
			}
			var response Message
			err = json.Unmarshal([]byte(msg.Payload), &response)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal response")
				cancel()
				return
			}
			retch <- &response
		}
	}()
	return retch
}

//func (m *Morpheus) Message(ctx context.Context, channel, from, to, route string, meta map[string]string, payload interface{}, responseTo *string, retch chan *Message) {
//	msgId := randomId()
//	message := Message{
//		Timestamp: time.Now().Unix(),
//		From:      from,
//		To:        to,
//		Payload:   payload,
//		Channel:   channel,
//		MsgId:     msgId,
//		Route:     route,
//		Meta:      meta,
//	}
//	if responseTo == nil {
//		message.ResponseChannel = fmt.Sprintf("%s:response:%s", channel, message.MsgId)
//	}
//	jMsg, err := json.Marshal(message)
//	if err != nil {
//		log.Error().Err(err).Msg("failed to marshal message")
//		return
//	}
//	if responseTo == nil {
//		sub := m.client.Subscribe(ctx, message.ResponseChannel)
//		go func() {
//			msg, err := sub.ReceiveMessage(ctx)
//			if err, ok := err.(*net.OpError); ok {
//				return
//			} else if err != nil {
//				log.Error().Err(err).Msg("failed to receive message")
//			}
//			var response Message
//			err = json.Unmarshal([]byte(msg.Payload), &response)
//			if err != nil {
//				log.Error().Err(err).Msg("failed to unmarshal response")
//				return
//			}
//			retch <- &response
//			_ = sub.Close()
//		}()
//	}
//	m.client.Publish(ctx, channel, string(jMsg))
//}

func (m *Morpheus) FetchService(name string, id string) (Service, error) {
	key := fmt.Sprintf("morpheus:service:%s:%s:health", name, id)
	val := m.client.Get(m.context, key)
	if val.Err() != nil {
		return Service{}, val.Err()
	}
	var svc Service
	err := json.Unmarshal([]byte(val.Val()), &svc)
	if err != nil {
		return Service{}, err
	}
	return svc, nil
}

func (m *Morpheus) ResolveService(name, path string) (*Service, error) {
	out := make([]Service, 0)
	for _, svc := range m.ListServices() {
		if svc.Routes.Match(path) && svc.Name == name {
			out = append(out, svc)
		}
	}
	if len(out) > 0 {
		idx := rand.Int() % len(out)
		return &out[idx], nil
	}
	return nil, fmt.Errorf("service not found")
}
func (m *Morpheus) RespondWithMeta(ctx context.Context, msg *Message, payload interface{}, meta map[string]string) {
	reply := *msg
	reply.Timestamp = time.Now().Unix()
	reply.MsgId = randomId()
	reply.Channel = msg.ResponseChannel
	reply.Payload = payload
	reply.From = msg.To
	reply.To = msg.From
	reply.Meta = meta
	m.Send(ctx, reply)
}
func (m *Morpheus) Respond(ctx context.Context, msg *Message, payload interface{}) {
	m.RespondWithMeta(ctx, msg, payload, nil)
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
