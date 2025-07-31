package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// Mock implementations for testing

type MockPocketBaseManager struct {
	mock.Mock
}

func (m *MockPocketBaseManager) DownloadPocketBase(version string, projectName string) error {
	args := m.Called(version, projectName)
	return args.Error(0)
}

func (m *MockPocketBaseManager) ExtractPocketBase(projectName string, version string) error {
	args := m.Called(projectName, version)
	return args.Error(0)
}

func (m *MockPocketBaseManager) SetPermissions(projectName string) error {
	args := m.Called(projectName)
	return args.Error(0)
}

func (m *MockPocketBaseManager) CreateSuperUser(projectName string, email string, password string) error {
	args := m.Called(projectName, email, password)
	return args.Error(0)
}

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

// Helper function to create a test record
func createTestRecord(id string, name string, port int, version string, subdomain string, status string) *pbmodels.Record {
	record := &pbmodels.Record{}
	record.Id = id
	record.Set("name", name)
	record.Set("port", port)
	record.Set("version", version)
	record.Set("subdomain", subdomain)
	record.Set("status", status)
	record.Set("created", "2023-01-01 00:00:00.000Z")
	record.Set("updated", "2023-01-01 00:00:00.000Z")
	return record
}

func TestOnServiceCreate(t *testing.T) {
	// Create mocks
	mockPBManager := new(MockPocketBaseManager)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)
	mockValidator := new(MockValidationService)
	mockConfigSync := new(MockConfigSyncService)

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "pockestrator-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create service hooks
	hooks := NewServiceHooks(
		mockConfigSync,
		mockPBManager,
		mockSystemdManager,
		mockCaddyManager,
		mockValidator,
		tempDir,
		"admin@example.com",
		"password123",
	)

	// Create test record
	record := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "creating")

	// Set up expectations
	mockPBManager.On("DownloadPocketBase", "0.20.0", "testservice").Return(nil)
	mockPBManager.On("ExtractPocketBase", "testservice", "0.20.0").Return(nil)
	mockPBManager.On("SetPermissions", "testservice").Return(nil)
	mockSystemdManager.On("CreateService", mock.AnythingOfType("*models.Service")).Return(nil)
	mockSystemdManager.On("EnableService", "testservice").Return(nil)
	mockCaddyManager.On("AddConfiguration", mock.AnythingOfType("*models.Service")).Return(nil)
	mockSystemdManager.On("StartService", "testservice").Return(nil)
	mockPBManager.On("CreateSuperUser", "testservice", "admin@example.com", "password123").Return(nil)

	// Create event
	event := &core.RecordEvent{
		Record: record,
	}

	// Call the hook
	err = hooks.onServiceCreate(event)

	// Verify expectations
	assert.NoError(t, err)
	mockPBManager.AssertExpectations(t)
	mockSystemdManager.AssertExpectations(t)
	mockCaddyManager.AssertExpectations(t)

	// Check that directories were created
	dirs := []string{
		filepath.Join(tempDir, "testservice", "pb_data"),
		filepath.Join(tempDir, "testservice", "pb_public"),
		filepath.Join(tempDir, "testservice", "pb_migrations"),
		filepath.Join(tempDir, "testservice", "pb_hooks"),
	}

	for _, dir := range dirs {
		_, err := os.Stat(dir)
		assert.NoError(t, err, "Directory should exist: %s", dir)
	}

	// Check that status was updated to "running"
	assert.Equal(t, "running", record.GetString("status"))
}

func TestOnServiceUpdate(t *testing.T) {
	// Create mocks
	mockPBManager := new(MockPocketBaseManager)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)
	mockValidator := new(MockValidationService)
	mockConfigSync := new(MockConfigSyncService)

	// Create service hooks
	hooks := NewServiceHooks(
		mockConfigSync,
		mockPBManager,
		mockSystemdManager,
		mockCaddyManager,
		mockValidator,
		"/tmp",
		"admin@example.com",
		"password123",
	)

	// Test case 1: Port change
	t.Run("PortChange", func(t *testing.T) {
		// Create original and updated records
		originalRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "running")
		updatedRecord := createTestRecord("test123", "testservice", 8092, "0.20.0", "test", "running")

		// Set up expectations
		mockSystemdManager.On("CreateService", mock.AnythingOfType("*models.Service")).Return(nil).Once()
		mockCaddyManager.On("AddConfiguration", mock.AnythingOfType("*models.Service")).Return(nil).Once()
		mockSystemdManager.On("StopService", "testservice").Return(nil).Once()
		mockSystemdManager.On("StartService", "testservice").Return(nil).Once()

		// Create event
		event := &core.RecordEvent{
			Record:         updatedRecord,
			OriginalRecord: originalRecord,
		}

		// Call the hook
		err := hooks.onServiceUpdate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockSystemdManager.AssertExpectations(t)
		mockCaddyManager.AssertExpectations(t)
	})

	// Test case 2: Subdomain change
	t.Run("SubdomainChange", func(t *testing.T) {
		// Create original and updated records
		originalRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "running")
		updatedRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "newtest", "running")

		// Set up expectations
		mockCaddyManager.On("AddConfiguration", mock.AnythingOfType("*models.Service")).Return(nil).Once()
		mockSystemdManager.On("StopService", "testservice").Return(nil).Once()
		mockSystemdManager.On("StartService", "testservice").Return(nil).Once()

		// Create event
		event := &core.RecordEvent{
			Record:         updatedRecord,
			OriginalRecord: originalRecord,
		}

		// Call the hook
		err := hooks.onServiceUpdate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockCaddyManager.AssertExpectations(t)
		mockSystemdManager.AssertExpectations(t)
	})

	// Test case 3: Status change (stopped to running)
	t.Run("StatusChangeToRunning", func(t *testing.T) {
		// Create original and updated records
		originalRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "stopped")
		updatedRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "running")

		// Set up expectations
		mockSystemdManager.On("StartService", "testservice").Return(nil).Once()

		// Create event
		event := &core.RecordEvent{
			Record:         updatedRecord,
			OriginalRecord: originalRecord,
		}

		// Call the hook
		err := hooks.onServiceUpdate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockSystemdManager.AssertExpectations(t)
	})

	// Test case 4: Status change (running to stopped)
	t.Run("StatusChangeToStopped", func(t *testing.T) {
		// Create original and updated records
		originalRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "running")
		updatedRecord := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "stopped")

		// Set up expectations
		mockSystemdManager.On("StopService", "testservice").Return(nil).Once()

		// Create event
		event := &core.RecordEvent{
			Record:         updatedRecord,
			OriginalRecord: originalRecord,
		}

		// Call the hook
		err := hooks.onServiceUpdate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockSystemdManager.AssertExpectations(t)
	})
}

func TestOnServiceDelete(t *testing.T) {
	// Create mocks
	mockPBManager := new(MockPocketBaseManager)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)
	mockValidator := new(MockValidationService)
	mockConfigSync := new(MockConfigSyncService)

	// Create service hooks
	hooks := NewServiceHooks(
		mockConfigSync,
		mockPBManager,
		mockSystemdManager,
		mockCaddyManager,
		mockValidator,
		"/tmp",
		"admin@example.com",
		"password123",
	)

	// Create test record
	record := createTestRecord("test123", "testservice", 8091, "0.20.0", "test", "running")

	// Set up expectations
	mockSystemdManager.On("RemoveService", "testservice").Return(nil)
	mockCaddyManager.On("RemoveConfiguration", "testservice").Return(nil)

	// Create event
	event := &core.RecordEvent{
		Record: record,
	}

	// Call the hook
	err := hooks.onServiceDelete(event)

	// Verify expectations
	assert.NoError(t, err)
	mockSystemdManager.AssertExpectations(t)
	mockCaddyManager.AssertExpectations(t)
}

func TestValidateBeforeCreate(t *testing.T) {
	// Create mocks
	mockPBManager := new(MockPocketBaseManager)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)
	mockValidator := new(MockValidationService)
	mockConfigSync := new(MockConfigSyncService)

	// Create service hooks
	hooks := NewServiceHooks(
		mockConfigSync,
		mockPBManager,
		mockSystemdManager,
		mockCaddyManager,
		mockValidator,
		"/tmp",
		"admin@example.com",
		"password123",
	)

	// Test case 1: Valid service with specified port
	t.Run("ValidServiceWithPort", func(t *testing.T) {
		// Create test record
		record := createTestRecord("", "testservice", 8091, "0.20.0", "test", "")

		// Set up expectations
		mockValidator.On("ValidateNameAvailable", "testservice").Return(nil).Once()
		mockValidator.On("ValidatePortAvailable", 8091, "").Return(nil).Once()

		// Create event
		event := &core.RecordCreateEvent{
			Record: record,
		}

		// Call the hook
		err := hooks.validateBeforeCreate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockValidator.AssertExpectations(t)
		assert.Equal(t, "creating", record.GetString("status"))
	})

	// Test case 2: Valid service with auto-assigned port
	t.Run("ValidServiceWithAutoPort", func(t *testing.T) {
		// Create test record
		record := createTestRecord("", "testservice", 0, "0.20.0", "test", "")

		// Set up expectations
		mockValidator.On("ValidateNameAvailable", "testservice").Return(nil).Once()
		mockValidator.On("GetNextAvailablePort").Return(8092, nil).Once()

		// Create event
		event := &core.RecordCreateEvent{
			Record: record,
		}

		// Call the hook
		err := hooks.validateBeforeCreate(event)

		// Verify expectations
		assert.NoError(t, err)
		mockValidator.AssertExpectations(t)
		assert.Equal(t, 8092, record.GetInt("port"))
		assert.Equal(t, "creating", record.GetString("status"))
	})
}
