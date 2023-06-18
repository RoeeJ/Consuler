package main

import "testing"
import "github.com/houqp/gtest"

type ConsulTests struct{}

var ci *ConsulInstance

func (s *ConsulTests) Setup(t *testing.T) {
	initializeLogger()
	if ci != nil {
		t.Fatal("consul already initialized")
	}
	_ci, err := NewInstance("test", "localhost:8500", 8500, "http")
	if err != nil {
		t.Fatal(err)
	}
	ci = _ci
	if err != nil {
		t.Fatal(err)
	}
}
func (s *ConsulTests) Teardown(t *testing.T) {
	if ci == nil {
		t.Fatal("consul not initialized")
	}
	ci = nil
}

// BeforeEach and AfterEach are invoked per test run
func (s *ConsulTests) BeforeEach(t *testing.T) {}
func (s *ConsulTests) AfterEach(t *testing.T)  {}

func TestConsul(t *testing.T) {
	gtest.RunSubTests(t, &ConsulTests{})
}

func (s *ConsulTests) SubTestServiceReg(t *testing.T) {
	svc, err := ci.NewService("test", "127.0.0.1", 8080)
	if err != nil {
		t.Fatal("failed to create service", err)
	}
	err = ci.UpdateService(svc)
	if err != nil {
		t.Fatal("failed to update service", err)
	}
	t.Logf("%+v", svc)
}
func (s *ConsulTests) SubTestServiceInvalidReg(t *testing.T) {
	_, err := ci.NewService("test", "", 0)
	if err == nil {
		t.Fatal("Invalid service registration should fail")
	}
}
