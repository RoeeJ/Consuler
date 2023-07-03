package morpheus

import (
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Timestamp       int64             `json:"timestamp,omitempty"`
	MsgId           string            `json:"msg_id,omitempty"`
	ResponseChannel string            `json:"response_channel,omitempty"`
	Channel         string            `json:"channel,omitempty"`
	Route           string            `json:"route,omitempty"`
	From            string            `json:"from,omitempty"`
	To              string            `json:"to,omitempty"`
	Payload         interface{}       `json:"payload,omitempty"`
	Err             error             `json:"err,omitempty"`
	Meta            map[string]string `json:"meta,omitempty"`
}

func FromJson(b []byte) *Message {
	var m *Message
	err := json.Unmarshal(b, &m)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal message")
		return nil
	}
	return m
}

func FromService(from string, service Service, route string, payload interface{}) Message {
	return FromServiceWithMeta(from, service, route, payload, nil)
}
func FromServiceWithMeta(from string, service Service, route string, payload interface{}, meta map[string]string) Message {
	msgId := randomId()
	return Message{
		MsgId:           msgId,
		ResponseChannel: fmt.Sprintf("%s:response:%s", service.Key(), msgId),
		From:            from,
		To:              service.Key(),
		Channel:         service.Key(),
		Route:           route,
		Payload:         payload,
		Meta:            meta,
	}
}
func FromRedisMessage(msg *redis.Message) Message {
	var m Message
	err := json.Unmarshal([]byte(msg.Payload), &m)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal response")
	}
	return m
}
func (m *Message) Json() []byte {
	b, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal message")
	}
	return b
}
