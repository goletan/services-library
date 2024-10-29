// /services/registry_test.go
package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/goletan/services"
)

// MockService implementing Service interface for testing
type MockService struct {
	name        string
	initialized bool
	started     bool
	shouldFail  bool
	failOnStart bool
	failOnInit  bool
	initDelay   time.Duration
	startDelay  time.Duration
	stopDelay   time.Duration
}

func (ms *MockService) Name() string {
	return ms.name
}

func (ms *MockService) Initialize() error {
	if ms.failOnInit {
		return errors.New("initialization failed")
	}
	time.Sleep(ms.initDelay) // Simulate initialization delay
	ms.initialized = true
	return nil
}

func (ms *MockService) Start() error {
	if ms.failOnStart {
		return errors.New("start failed")
	}
	time.Sleep(ms.startDelay) // Simulate start delay
	ms.started = true
	return nil
}

func (ms *MockService) Stop() error {
	time.Sleep(ms.stopDelay) // Simulate stop delay
	ms.started = false
	return nil
}

func TestServiceRegistry(t *testing.T) {
	registry := services.NewRegistry()

	mockService := &MockService{name: "TestService"}
	err := registry.RegisterService(mockService)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Test InitializeAll
	err = registry.InitializeAll()
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}

	if !mockService.initialized {
		t.Errorf("Expected service to be initialized, but it wasn't")
	}

	// Test StartAll
	err = registry.StartAll()
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}

	if !mockService.started {
		t.Errorf("Expected service to be started, but it wasn't")
	}

	// Test StopAll
	err = registry.StopAll()
	if err != nil {
		t.Fatalf("Failed to stop services: %v", err)
	}

	if mockService.started {
		t.Errorf("Expected service to be stopped, but it was still running")
	}
}

func TestServiceRegistryWithFailures(t *testing.T) {
	registry := services.NewRegistry()

	// MockService that fails on initialization
	failingInitService := &MockService{name: "FailInitService", failOnInit: true}
	err := registry.RegisterService(failingInitService)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Test InitializeAll expecting failure
	err = registry.InitializeAll()
	if err == nil {
		t.Errorf("Expected initialization to fail, but it succeeded")
	}

	// MockService that fails on start
	failingStartService := &MockService{name: "FailStartService", failOnStart: true}
	err = registry.RegisterService(failingStartService)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Initialize should work even if start fails
	err = registry.InitializeAll()
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}

	// Test StartAll expecting failure
	err = registry.StartAll()
	if err == nil {
		t.Errorf("Expected start to fail, but it succeeded")
	}
}

func TestServiceRegistryWithDelays(t *testing.T) {
	registry := services.NewRegistry()

	// MockService with delays in each lifecycle method
	delayedService := &MockService{
		name:       "DelayedService",
		initDelay:  100 * time.Millisecond,
		startDelay: 100 * time.Millisecond,
		stopDelay:  100 * time.Millisecond,
	}
	err := registry.RegisterService(delayedService)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Test InitializeAll with delays
	startTime := time.Now()
	err = registry.InitializeAll()
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}
	if time.Since(startTime) < 100*time.Millisecond {
		t.Errorf("Initialization completed too quickly, expected a delay")
	}

	// Test StartAll with delays
	startTime = time.Now()
	err = registry.StartAll()
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	if time.Since(startTime) < 100*time.Millisecond {
		t.Errorf("Start completed too quickly, expected a delay")
	}

	// Test StopAll with delays
	startTime = time.Now()
	err = registry.StopAll()
	if err != nil {
		t.Fatalf("Failed to stop services: %v", err)
	}
	if time.Since(startTime) < 100*time.Millisecond {
		t.Errorf("Stop completed too quickly, expected a delay")
	}
}
