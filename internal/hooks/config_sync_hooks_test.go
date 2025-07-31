package hooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

func TestConfigConflictMiddleware(t *testing.T) {
	// Create mock config sync service
	mockConfigSync := new(MockConfigSyncService)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)
	mockValidator := new(MockValidationService)
	mockServiceRepo := new(MockServiceRepository)

	// Create config sync hooks
	hooks := NewConfigSyncHooks(
		mockConfigSync,
		mockSystemdManager,
		mockCaddyManager,
		mockValidator,
		mockServiceRepo,
		true,
	)

	// Create test handler
	handler := func(c *apis.ApiContext) error {
		// Set config conflict in context
		conflict := &services.ConfigConflict{
			ServiceID:      "test123",
			ServiceName:    "testservice",
			HasConflict:    true,
			ConflictFields: map[string]string{"port": "Database: 8091, Systemd: 8092"},
			SystemState: &models.Service{
				ID:        "test123",
				Name:      "testservice",
				Port:      8092,
				Version:   "0.20.0",
				Subdomain: "test",
				Status:    "running",
			},
		}
		c.Set("config_conflict", conflict)
		return c.JSON(http.StatusOK, map[string]interface{}{"message": "test"})
	}

	// Create middleware handler
	middlewareHandler := hooks.configConflictMiddleware(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Create API context
	c := apis.NewApiContext(req, rec, nil)

	// Call middleware
	err := middlewareHandler(c)

	// Check result
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// Mock implementations for testing

// MockConfigSyncService is a mock implementation of services.ConfigSyncService
type MockConfigSyncService struct {
	mock.Mock
}

func (m *MockConfigSyncService) SyncServiceToSystem(service *models.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockConfigSyncService) SyncSystemToService(service *models.Service) (*models.Service, error) {
	args := m.Called(service)
	return args.Get(0).(*models.Service), args.Error(1)
}

func (m *MockConfigSyncService) DetectConflicts(service *models.Service) (*services.ConfigConflict, error) {
	args := m.Called(service)
	return args.Get(0).(*services.ConfigConflict), args.Error(1)
}

func (m *MockConfigSyncService) ValidateServiceConfig(service *models.Service) (bool, []string) {
	args := m.Called(service)
	return args.Bool(0), args.Get(1).([]string)
}

// MockSystemdManager is a mock implementation of services.SystemdManager
type MockSystemdManager struct {
	mock.Mock
}

func (m *MockSystemdManager) CreateService(service *models.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockSystemdManager) EnableService(serviceName string) error {
	args := m.Called(serviceName)
	return args.Error(0)
}

func (m *MockSystemdManager) StartService(serviceName string) error {
	args := m.Called(serviceName)
	return args.Error(0)
}

func (m *MockSystemdManager) StopService(serviceName string) error {
	args := m.Called(serviceName)
	return args.Error(0)
}

func (m *MockSystemdManager) RemoveService(serviceName string) error {
	args := m.Called(serviceName)
	return args.Error(0)
}

func (m *MockSystemdManager) IsServiceRunning(serviceName string) bool {
	args := m.Called(serviceName)
	return args.Bool(0)
}

func (m *MockSystemdManager) GetServiceStatus(serviceName string) (*services.ServiceStatus, error) {
	args := m.Called(serviceName)
	return args.Get(0).(*services.ServiceStatus), args.Error(1)
}

// MockCaddyManager is a mock implementation of services.CaddyManager
type MockCaddyManager struct {
	mock.Mock
}

func (m *MockCaddyManager) AddConfiguration(service *models.Service) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockCaddyManager) RemoveConfiguration(serviceName string) error {
	args := m.Called(serviceName)
	return args.Error(0)
}

func (m *MockCaddyManager) ReloadCaddy() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCaddyManager) ValidateConfiguration(service *models.Service) bool {
	args := m.Called(service)
	return args.Bool(0)
}

// MockValidationService is a mock implementation of services.ValidationService
type MockValidationService struct {
	mock.Mock
}

func (m *MockValidationService) ValidateService(service *models.Service) (*models.ServiceValidation, error) {
	args := m.Called(service)
	return args.Get(0).(*models.ServiceValidation), args.Error(1)
}

func (m *MockValidationService) ValidatePortAvailable(port int, excludeService string) error {
	args := m.Called(port, excludeService)
	return args.Error(0)
}

func (m *MockValidationService) ValidateNameAvailable(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockValidationService) GetNextAvailablePort() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// MockServiceRepository is a mock implementation of ServiceRepository
type MockServiceRepository struct {
	mock.Mock
}

func (m *MockServiceRepository) FindServiceByID(id string) (*models.Service, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Service), args.Error(1)
}

func (m *MockServiceRepository) FindServiceByName(name string) (*models.Service, error) {
	args := m.Called(name)
	return args.Get(0).(*models.Service), args.Error(1)
}

func (m *MockServiceRepository) ListAllServices() ([]*models.Service, error) {
	args := m.Called()
	return args.Get(0).([]*models.Service), args.Error(1)
}

func (m *MockServiceRepository) UpdateService(service *models.Service) error {
	args := m.Called(service)
	return args.Error(0)
}
