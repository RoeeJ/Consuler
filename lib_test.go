package morpheus

import (
	"github.com/roeej/morpheus/logging"
	"github.com/rs/zerolog/log"
	"testing"
	"time"
)

var m, err = Init()

func Setup() {
	if err != nil {
		panic(err)
	}
	m.FlushDB()
}
func Teardown() {
	svcs := m.ListServices()

	for _, svc := range svcs {
		m.Services.Remove(svc)
	}

}

func TestMain(m *testing.M) {
	logging.InitLogger()
	Setup()
	m.Run()
	Teardown()
}

func TestInit(t *testing.T) {
	if m.client == nil {
		t.Errorf("expected client to be initialized")
	}
}

func TestMorpheus_Connect(t *testing.T) {
	if m.client == nil {
		t.Errorf("expected client to be initialized")
	}
	if err := m.Connect(); err != nil {
		t.Errorf("expected no error, got %s", err)
	}
}

func TestMorpheus_RegisterService(t *testing.T) {
	outch := make(chan bool)
	name := "test"
	port := 0
	routes := []string{"/test"}
	svc, err := m.RegisterService(name, port, routes, func(morpheus *Morpheus, msg *Message) {
		log.Info().Interface("msg", msg).Msg("received message")
		close(outch)
	})
	m.Message(nil, svc.Key(), Message{
		Timestamp: time.Now().Unix(),
		From:      "client:/",
		To:        svc.Key(),
		Payload:   "Testing",
		Route:     "/test",
	})
	select {
	case <-outch:
		break
	case <-time.After(5 * time.Second):
		t.Errorf("expected message to be received")
	}
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
}

func TestMorpheus_ListServices(t *testing.T) {
	svcs := m.ListServices()
	internal := m.internalListServices()
	if len(svcs) != internal {
		t.Errorf("expected %d service, got %d", internal, len(svcs))
	}
}
func TestMorpheus_UpdateRoutes(t *testing.T) {
	svcs := m.ListServices()
	for _, svc := range svcs {
		svc.Routes = map[string]bool{"/test": true}
		m.UpdateRoutes(&svc)
	}
}
