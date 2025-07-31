package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tigawanna/pockestrator/internal/models"
)

// MockServiceRepository implements the ServiceRepository interface for testing
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

func (m *MockServiceRepository) UpdateService(service *models.Service) error {
	m.services[service.ID] = service
	return nil
}

func (m *MockServiceRepository) AddService(service *models.Service) {
	m.services[service.ID] = service
}

// MockSystemdManager implements the SystemdManager interface for testing
type MockSystemdManager struct {
	services map[string]*ServiceStatus
	files    map[string]string
}

func NewMockSystemdManager() *MockSystemdManager {
	return &MockSystemdManager{
		services: make(map[string]*ServiceStatus),
		files:    make(map[string]string),
	}
}

func (m *MockSystemdManager) CreateService(service *models.Service) error {
	m.files[service.Name] = generateMockSystemdFile(service)
	return nil
}

func (m *MockSystemdManager) EnableService(serviceName string) error {
	if status, exists := m.services[serviceName]; exists {
		status.Enabled = true
	} else {
		m.services[serviceName] = &ServiceStatus{
			Name:    serviceName,
			Enabled: true,
			Active:  false,
			Status:  "inactive",
		}
	}
	return nil
}

func (m *MockSystemdManager) StartService(serviceName string) error {
	if status, exists := m.services[serviceName]; exists {
		status.Active = true
		status.Status = "active"
	} else {
		m.services[serviceName] = &ServiceStatus{
			Name:    serviceName,
			Enabled: false,
			Active:  true,
			Status:  "active",
		}
	}
	return nil
}

func (m *MockSystemdManager) StopService(serviceName string) error {
	if status, exists := m.services[serviceName]; exists {
		status.Active = false
		status.Status = "inactive"
	}
	return nil
}

func (m *MockSystemdManager) RemoveService(serviceName string) error {
	delete(m.services, serviceName)
	delete(m.files, serviceName)
	return nil
}

func (m *MockSystemdManager) IsServiceRunning(serviceName string) bool {
	if status, exists := m.services[serviceName]; exists {
		return status.Active
	}
	return false
}

func (m *MockSystemdManager) GetServiceStatus(serviceName string) (*ServiceStatus, error) {
	if status, exists := m.services[serviceName]; exists {
		return status, nil
	}
	return &ServiceStatus{
		Name:    serviceName,
		Enabled: false,
		Active:  false,
		Status:  "not-found",
	}, nil
}

func (m *MockSystemdManager) GetServiceFile(serviceName string) string {
	return m.files[serviceName]
}

// MockCaddyManager implements the CaddyManager interface for testing
type MockCaddyManager struct {
	configs map[string]string
	domain  string
}

func NewMockCaddyManager(domain string) *MockCaddyManager {
	return &MockCaddyManager{
		configs: make(map[string]string),
		domain:  domain,
	}
}

func (m *MockCaddyManager) AddConfiguration(service *models.Service) error {
	m.configs[service.Name] = generateMockCaddyConfig(service, m.domain)
	return nil
}

func (m *MockCaddyManager) RemoveConfiguration(serviceName string) error {
	delete(m.configs, serviceName)
	return nil
}

func (m *MockCaddyManager) ReloadCaddy() error {
	return nil
}

func (m *MockCaddyManager) ValidateConfiguration(service *models.Service) bool {
	config, exists := m.configs[service.Name]
	if !exists {
		return false
	}
	return config != ""
}

func (m *MockCaddyManager) GetCaddyfile() string {
	var content string
	for _, config := range m.configs {
		content += config + "\n"
	}
	return content
}

// Helper functions for testing
func generateMockSystemdFile(service *models.Service) string {
	return `[Unit]
Description=PocketBase service for ` + service.Name + `
After=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu/` + service.Name + `
ExecStart=/home/ubuntu/` + service.Name + `/pocketbase serve --http=127.0.0.1:` + string(service.Port) + `
Restart=always
RestartSec=5
StandardOutput=append:/home/ubuntu/` + service.Name + `/service.log
StandardError=append:/home/ubuntu/` + service.Name + `/service.log

[Install]
WantedBy=multi-user.target`
}

func generateMockCaddyConfig(service *models.Service, domain string) string {
	return service.Subdomain + "." + domain + ` {
    reverse_proxy 127.0.0.1:` + string(service.Port) + `
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}
`
}

func setupTestEnvironment(t *testing.T) (string, *ConfigSyncServiceImpl, *MockServiceRepository, *MockSystemdManager, *MockCaddyManager) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-sync-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create subdirectories
	systemdDir := filepath.Join(tempDir, "systemd")
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		t.Fatalf("Failed to create systemd directory: %v", err)
	}

	// Create mock dependencies
	mockRepo := NewMockServiceRepository()
	mockSystemd := NewMockSystemdManager()
	mockCaddy := NewMockCaddyManager("example.com")

	// Create config sync service
	configSync := NewConfigSyncService(
		mockSystemd,
		mockCaddy,
		tempDir,
		systemdDir,
		filepath.Join(tempDir, "Caddyfile"),
		"example.com",
		mockRepo,
	)

	return tempDir, configSync, mockRepo, mockSystemd, mockCaddy
}

func TestDetectConflicts(t *testing.T) {
	tempDir, configSync, mockRepo, mockSystemd, mockCaddy := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test service
	service := &models.Service{
		ID:        "test1",
		Name:      "test-service",
		Port:      8091,
		Version:   "0.15.0",
		Subdomain: "test",
		Status:    "running",
	}

	// Add service to repository
	mockRepo.AddService(service)

	// Create systemd service with different port
	modifiedService := *service
	modifiedService.Port = 8092
	mockSystemd.CreateService(&modifiedService)

	// Create caddy config with different subdomain
	modifiedService = *service
	modifiedService.Subdomain = "test-modified"
	mockCaddy.AddConfiguration(&modifiedService)

	// Set service status to stopped
	mockSystemd.StopService(service.Name)

	// Detect conflicts
	conflict, err := configSync.DetectConflicts(service)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	// Verify conflicts were detected
	if !conflict.HasConflict {
		t.Errorf("Expected conflicts but none were detected")
	}

	// Check specific conflicts
	if _, exists := conflict.ConflictFields["port"]; !exists {
		t.Errorf("Expected port conflict but none was detected")
	}

	if _, exists := conflict.ConflictFields["subdomain"]; !exists {
		t.Errorf("Expected subdomain conflict but none was detected")
	}

	if _, exists := conflict.ConflictFields["status"]; !exists {
		t.Errorf("Expected status conflict but none was detected")
	}
}

func TestSyncServiceToSystem(t *testing.T) {
	tempDir, configSync, mockRepo, mockSystemd, mockCaddy := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test service
	service := &models.Service{
		ID:        "test1",
		Name:      "test-service",
		Port:      8091,
		Version:   "0.15.0",
		Subdomain: "test",
		Status:    "running",
	}

	// Add service to repository
	mockRepo.AddService(service)

	// Sync service to system
	err := configSync.SyncServiceToSystem(service)
	if err != nil {
		t.Fatalf("SyncServiceToSystem failed: %v", err)
	}

	// Verify systemd service was created
	if !mockSystemd.ValidateConfiguration(service) {
		t.Errorf("Expected systemd service to be created")
	}

	// Verify caddy config was created
	if !mockCaddy.ValidateConfiguration(service) {
		t.Errorf("Expected caddy config to be created")
	}
}

func TestSyncSystemToService(t *testing.T) {
	tempDir, configSync, mockRepo, mockSystemd, mockCaddy := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test service
	service := &models.Service{
		ID:        "test1",
		Name:      "test-service",
		Port:      8091,
		Version:   "0.15.0",
		Subdomain: "test",
		Status:    "running",
	}

	// Add service to repository
	mockRepo.AddService(service)

	// Create systemd service with different port
	modifiedService := *service
	modifiedService.Port = 8092
	mockSystemd.CreateService(&modifiedService)

	// Create caddy config with different subdomain
	modifiedService = *service
	modifiedService.Subdomain = "test-modified"
	mockCaddy.AddConfiguration(&modifiedService)

	// Set service status to stopped
	mockSystemd.StopService(service.Name)

	// Sync system to service
	updatedService, err := configSync.SyncSystemToService(service)
	if err != nil {
		t.Fatalf("SyncSystemToService failed: %v", err)
	}

	// Verify service was updated with system values
	if updatedService.Port != 8092 {
		t.Errorf("Expected port to be updated to 8092, got %d", updatedService.Port)
	}

	if updatedService.Subdomain != "test-modified" {
		t.Errorf("Expected subdomain to be updated to test-modified, got %s", updatedService.Subdomain)
	}

	if updatedService.Status != "stopped" {
		t.Errorf("Expected status to be updated to stopped, got %s", updatedService.Status)
	}
}

func TestValidateServiceConfig(t *testing.T) {
	tempDir, configSync, mockRepo, mockSystemd, mockCaddy := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test service
	service := &models.Service{
		ID:        "test1",
		Name:      "test-service",
		Port:      8091,
		Version:   "0.15.0",
		Subdomain: "test",
		Status:    "running",
	}

	// Add service to repository
	mockRepo.AddService(service)

	// Create matching systemd and caddy configs
	mockSystemd.CreateService(service)
	mockCaddy.AddConfiguration(service)

	// Create PocketBase binary directory and file
	servicePath := filepath.Join(tempDir, service.Name)
	if err := os.MkdirAll(servicePath, 0755); err != nil {
		t.Fatalf("Failed to create service directory: %v", err)
	}
	binaryPath := filepath.Join(servicePath, "pocketbase")
	if err := os.WriteFile(binaryPath, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Validate service config
	isValid, issues := configSync.ValidateServiceConfig(service)
	if !isValid {
		t.Errorf("Expected service config to be valid, but got issues: %v", issues)
	}
	if len(issues) > 0 {
		t.Errorf("Expected no issues, got: %v", issues)
	}

	// Modify systemd config to create a conflict
	modifiedService := *service
	modifiedService.Port = 8092
	mockSystemd.CreateService(&modifiedService)

	// Validate again
	isValid, issues = configSync.ValidateServiceConfig(service)
	if isValid {
		t.Errorf("Expected service config to be invalid")
	}
	if len(issues) == 0 {
		t.Errorf("Expected issues to be reported")
	}
}
