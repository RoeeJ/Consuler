package message

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

type Message struct {
	Timestamp       int64               `json:"timestamp,omitempty"`
	MsgId           string              `json:"msg_id,omitempty"`
	ResponseChannel string              `json:"response_channel,omitempty"`
	Channel         string              `json:"channel,omitempty"`
	Route           string              `json:"route,omitempty"`
	From            string              `json:"from,omitempty"`
	To              string              `json:"to,omitempty"`
	Payload         interface{}         `json:"payload,omitempty"`
	Err             error               `json:"err,omitempty"`
	Meta            map[string][]string `json:"meta,omitempty"`
}

func FromNatsMsg(msg *nats.Msg) (*Message, error) {
	var m Message
	err := json.Unmarshal(msg.Data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
func FromNatsRequest(req micro.Request) (*Message, error) {
	var m Message
	err := json.Unmarshal(req.Data(), &m)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers() {
		m.Meta[k] = v
	}
	return &m, nil
}
func (m *Message) ToNatsMsg() (*nats.Msg, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return &nats.Msg{Data: data}, nil
}

func (m *Message) JSON() []byte {
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return data
}
