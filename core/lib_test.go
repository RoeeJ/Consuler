package morpheus

import (
	"github.com/roeej/morpheus/logging"
	"testing"
)

var m, err = Init(Options{Mock: true})

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
