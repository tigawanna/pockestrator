package validators

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// MockServiceRepository is a mock implementation of ServiceRepository
type MockServiceRepository struct {
	services map[string]*models.Service
}

func NewMockServiceRepository() *MockServiceRepository {
	return &MockServiceRepository{
		services: make(map[string]*models.Service),
	}
}

func (m *MockServiceRepository) FindServiceByName(name string) (*models.Service, error) {
	for _, service := range m.services {
		if service.Name == name {
			return service, nil
		}
	}
	return nil, nil
}

func (m *MockServiceRepository) FindServiceByPort(port int) (*models.Service, error) {
	for _, service := range m.services {
		if service.Port == port {
			return service, nil
		}
	}
	return nil, nil
}

func (m *MockServiceRepository) ListAllServices() ([]*models.Service, error) {
	services := make([]*models.Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}
	return services, nil
}

func (m *MockServiceRepository) AddService(service *models.Service) {
	m.services[service.ID] = service
}

// MockSystemdManager is a mock implementation of SystemdManager
type MockSystemdManager struct {
	runningServices map[string]bool
	enabledServices map[string]bool
}

func NewMockSystemdManager() *MockSystemdManager {
	return &MockSystemdManager{
		runningServices: make(map[string]bool),
		enabledServices: make(map[string]bool),
	}
}

func (m *MockSystemdManager) CreateService(service *models.Service) error {
	return nil
}

func (m *MockSystemdManager) EnableService(serviceName string) error {
	m.enabledServices[serviceName] = true
	return nil
}

func (m *MockSystemdManager) StartService(serviceName string) error {
	m.runningServices[serviceName] = true
	return nil
}

func (m *MockSystemdManager) StopService(serviceName string) error {
	m.runningServices[serviceName] = false
	return nil
}

func (m *MockSystemdManager) RemoveService(serviceName string) error {
	delete(m.runningServices, serviceName)
	delete(m.enabledServices, serviceName)
	return nil
}

func (m *MockSystemdManager) IsServiceRunning(serviceName string) bool {
	return m.runningServices[serviceName]
}

func (m *MockSystemdManager) GetServiceStatus(serviceName string) (*services.ServiceStatus, error) {
	if _, exists := m.runningServices[serviceName]; !exists && _, enabled := m.enabledServices[serviceName]; !enabled {
		return nil, errors.New("service not found")
	}
	
	return &services.ServiceStatus{
		Name:    serviceName,
		Active:  m.runningServices[serviceName],
		Enabled: m.enabledServices[serviceName],
		Status:  m.runningServices[serviceName] ? "active" : "inactive",
	}, nil
}

// MockCaddyManager is a mock implementation of CaddyManager
type MockCaddyManager struct {
	configurations map[string]bool
}

func NewMockCaddyManager() *MockCaddyManager {
	return &MockCaddyManager{
		configurations: make(map[string]bool),
	}
}

func (m *MockCaddyManager) AddConfiguration(service *models.Service) error {
	m.configurations[service.Name] = true
	return nil
}

func (m *MockCaddyManager) RemoveConfiguration(serviceName string) error {
	delete(m.configurations, serviceName)
	return nil
}

func (m *MockCaddyManager) ReloadCaddy() error {
	return nil
}

func (m *MockCaddyManager) ValidateConfiguration(service *models.Service) bool {
	return m.configurations[service.Name]
}

// TestServiceValidator tests the ServiceValidator implementation
func TestServiceValidator(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock dependencies
	mockRepo := NewMockServiceRepository()
	mockSystemd := NewMockSystemdManager()
	mockCaddy := NewMockCaddyManager()

	// Create validator
	validator := NewServiceValidator(mockSystemd, mockCaddy, tempDir, mockRepo)

	// Test service for validation
	service := &models.Service{
		ID:        "test123",
		Name:      "testservice",
		Port:      8091,
		Version:   "0.20.0",
		Subdomain: "testservice",
		Status:    "running",
	}

	// Add service to repository
	mockRepo.AddService(service)

	// Test ValidateNameAvailable
	t.Run("ValidateNameAvailable", func(t *testing.T) {
		// Test with existing name
		err := validator.ValidateNameAvailable("testservice")
		if err == nil {
			t.Errorf("Expected error for existing service name, got nil")
		}

		// Test with new name
		err = validator.ValidateNameAvailable("newservice")
		if err != nil {
			t.Errorf("Expected no error for new service name, got: %v", err)
		}

		// Test with invalid name
		err = validator.ValidateNameAvailable("invalid name with spaces")
		if err == nil {
			t.Errorf("Expected error for invalid service name, got nil")
		}
	})

	// Test ValidatePortAvailable
	t.Run("ValidatePortAvailable", func(t *testing.T) {
		// Test with existing port
		err := validator.ValidatePortAvailable(8091, "")
		if err == nil {
			t.Errorf("Expected error for existing port, got nil")
		}

		// Test with existing port but excluded service
		err = validator.ValidatePortAvailable(8091, "testservice")
		if err != nil {
			t.Errorf("Expected no error when excluding the service with the port, got: %v", err)
		}

		// Test with new port
		err = validator.ValidatePortAvailable(8092, "")
		if err != nil {
			t.Errorf("Expected no error for new port, got: %v", err)
		}

		// Test with invalid port range
		err = validator.ValidatePortAvailable(7999, "")
		if err == nil {
			t.Errorf("Expected error for port below range, got nil")
		}

		err = validator.ValidatePortAvailable(10000, "")
		if err == nil {
			t.Errorf("Expected error for port above range, got nil")
		}
	})

	// Test GetNextAvailablePort
	t.Run("GetNextAvailablePort", func(t *testing.T) {
		// Add services with consecutive ports
		mockRepo.AddService(&models.Service{
			ID:   "test1",
			Name: "service1",
			Port: 8091,
		})
		mockRepo.AddService(&models.Service{
			ID:   "test2",
			Name: "service2",
			Port: 8092,
		})

		// Next port should be 8093
		port, err := validator.GetNextAvailablePort()
		if err != nil {
			t.Errorf("Failed to get next available port: %v", err)
		}
		if port != 8093 {
			t.Errorf("Expected next port to be 8093, got %d", port)
		}
	})

	// Test ValidateService
	t.Run("ValidateService", func(t *testing.T) {
		// Create test service directory and binary
		serviceDir := filepath.Join(tempDir, service.Name)
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			t.Fatalf("Failed to create service directory: %v", err)
		}
		
		binaryPath := filepath.Join(serviceDir, "pocketbase")
		if err := os.WriteFile(binaryPath, []byte("mock binary"), 0755); err != nil {
			t.Fatalf("Failed to create mock binary: %v", err)
		}

		// Test with no systemd or caddy configuration
		validation, err := validator.ValidateService(service)
		if err != nil {
			t.Errorf("Failed to validate service: %v", err)
		}
		
		if validation.BinaryExists != true {
			t.Errorf("Expected BinaryExists to be true, got false")
		}
		
		if validation.SystemdExists != false {
			t.Errorf("Expected SystemdExists to be false, got true")
		}
		
		if validation.CaddyConfigured != false {
			t.Errorf("Expected CaddyConfigured to be false, got true")
		}
		
		if len(validation.Issues) == 0 {
			t.Errorf("Expected validation issues, got none")
		}

		// Configure systemd and caddy
		mockSystemd.EnableService(service.Name)
		mockSystemd.StartService(service.Name)
		mockCaddy.AddConfiguration(service)

		// Test with proper configuration
		validation, err = validator.ValidateService(service)
		if err != nil {
			t.Errorf("Failed to validate service: %v", err)
		}
		
		if validation.BinaryExists != true {
			t.Errorf("Expected BinaryExists to be true, got false")
		}
		
		if validation.SystemdExists != true {
			t.Errorf("Expected SystemdExists to be true, got false")
		}
		
		if validation.SystemdRunning != true {
			t.Errorf("Expected SystemdRunning to be true, got false")
		}
		
		if validation.CaddyConfigured != true {
			t.Errorf("Expected CaddyConfigured to be true, got false")
		}
		
		if validation.PortMatches != true {
			t.Errorf("Expected PortMatches to be true, got false")
		}
	})
}