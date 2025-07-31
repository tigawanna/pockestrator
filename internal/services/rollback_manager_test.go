package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tigawanna/pockestrator/internal/models"
)

// Mock implementations for testing
type MockRollbackOperation struct {
	mock.Mock
}

func (m *MockRollbackOperation) Execute() error {
	args := m.Called()
	return args.Error(0)
}

func TestRollbackManager(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "rollback-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create rollback manager
	rm := NewRollbackManager(tempDir)

	// Test adding operations
	t.Run("AddOperation", func(t *testing.T) {
		// Add some operations
		rm.AddOperation("test", "Test operation 1", func() error {
			return nil
		})
		rm.AddOperation("test", "Test operation 2", func() error {
			return nil
		})

		// Check that operations were added
		assert.Equal(t, 2, len(rm.operations))
		assert.Equal(t, "Test operation 1", rm.operations[0].Description)
		assert.Equal(t, "Test operation 2", rm.operations[1].Description)
	})

	// Test successful rollback
	t.Run("SuccessfulRollback", func(t *testing.T) {
		// Create new rollback manager
		rm := NewRollbackManager(tempDir)

		// Create counters to track execution
		counter1 := 0
		counter2 := 0

		// Add operations
		rm.AddOperation("test", "Test operation 1", func() error {
			counter1++
			return nil
		})
		rm.AddOperation("test", "Test operation 2", func() error {
			counter2++
			return nil
		})

		// Execute rollback
		err := rm.Rollback()
		assert.NoError(t, err)

		// Check that operations were executed in reverse order
		assert.Equal(t, 1, counter1)
		assert.Equal(t, 1, counter2)

		// Check that operations were cleared
		assert.Equal(t, 0, len(rm.operations))
	})

	// Test rollback with errors
	t.Run("RollbackWithErrors", func(t *testing.T) {
		// Create new rollback manager
		rm := NewRollbackManager(tempDir)

		// Create counters to track execution
		counter1 := 0
		counter2 := 0

		// Add operations
		rm.AddOperation("test", "Test operation 1", func() error {
			counter1++
			return errors.New("operation 1 failed")
		})
		rm.AddOperation("test", "Test operation 2", func() error {
			counter2++
			return nil
		})

		// Execute rollback
		err := rm.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "errors occurred during rollback")

		// Check that all operations were executed despite errors
		assert.Equal(t, 1, counter1)
		assert.Equal(t, 1, counter2)

		// Check that operations were cleared
		assert.Equal(t, 0, len(rm.operations))
	})

	// Test clear operations
	t.Run("ClearOperations", func(t *testing.T) {
		// Create new rollback manager
		rm := NewRollbackManager(tempDir)

		// Add operations
		rm.AddOperation("test", "Test operation 1", func() error {
			return nil
		})
		rm.AddOperation("test", "Test operation 2", func() error {
			return nil
		})

		// Clear operations
		rm.Clear()

		// Check that operations were cleared
		assert.Equal(t, 0, len(rm.operations))
	})
}

func TestServiceCreationManager(t *testing.T) {
	// Create mocks
	mockPBManager := new(MockPocketBaseManager)
	mockSystemdManager := new(MockSystemdManager)
	mockCaddyManager := new(MockCaddyManager)

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "service-creation-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create service creation manager
	scm := NewServiceCreationManager(
		mockPBManager,
		mockSystemdManager,
		mockCaddyManager,
		tempDir,
		"admin@example.com",
		"password123",
	)

	// Test successful service creation
	t.Run("SuccessfulServiceCreation", func(t *testing.T) {
		// Create test service
		service := &models.Service{
			ID:        "test123",
			Name:      "testservice",
			Port:      8091,
			Version:   "0.20.0",
			Subdomain: "test",
			Status:    "creating",
		}

		// Set up expectations
		mockPBManager.On("DownloadPocketBase", "0.20.0", "testservice").Return(nil).Once()
		mockPBManager.On("ExtractPocketBase", "testservice", "0.20.0").Return(nil).Once()
		mockPBManager.On("SetPermissions", "testservice").Return(nil).Once()
		mockSystemdManager.On("CreateService", service).Return(nil).Once()
		mockSystemdManager.On("EnableService", "testservice").Return(nil).Once()
		mockCaddyManager.On("AddConfiguration", service).Return(nil).Once()
		mockSystemdManager.On("StartService", "testservice").Return(nil).Once()
		mockPBManager.On("CreateSuperUser", "testservice", "admin@example.com", "password123").Return(nil).Once()

		// Create service
		err := scm.CreateService(service)
		assert.NoError(t, err)

		// Verify expectations
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
	})

	// Test service creation with rollback
	t.Run("ServiceCreationWithRollback", func(t *testing.T) {
		// Create test service
		service := &models.Service{
			ID:        "test123",
			Name:      "testservice2",
			Port:      8092,
			Version:   "0.20.0",
			Subdomain: "test2",
			Status:    "creating",
		}

		// Set up expectations
		mockPBManager.On("DownloadPocketBase", "0.20.0", "testservice2").Return(nil).Once()
		mockPBManager.On("ExtractPocketBase", "testservice2", "0.20.0").Return(nil).Once()
		mockPBManager.On("SetPermissions", "testservice2").Return(nil).Once()
		mockSystemdManager.On("CreateService", service).Return(errors.New("systemd error")).Once()

		// Rollback expectations
		mockSystemdManager.On("RemoveService", "testservice2").Return(nil).Once()

		// Create service (should fail and trigger rollback)
		err := scm.CreateService(service)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create systemd service")

		// Verify expectations
		mockPBManager.AssertExpectations(t)
		mockSystemdManager.AssertExpectations(t)
	})

	// Test service update
	t.Run("ServiceUpdate", func(t *testing.T) {
		// Create test services
		oldService := &models.Service{
			ID:        "test123",
			Name:      "testservice3",
			Port:      8093,
			Version:   "0.20.0",
			Subdomain: "test3",
			Status:    "running",
		}

		newService := &models.Service{
			ID:        "test123",
			Name:      "testservice3",
			Port:      8094,
			Version:   "0.20.0",
			Subdomain: "test3-new",
			Status:    "running",
		}

		// Set up expectations
		mockSystemdManager.On("CreateService", newService).Return(nil).Once()
		mockCaddyManager.On("AddConfiguration", newService).Return(nil).Once()
		mockSystemdManager.On("StopService", "testservice3").Return(nil).Once()
		mockSystemdManager.On("StartService", "testservice3").Return(nil).Once()

		// Update service
		err := scm.UpdateService(newService, oldService)
		assert.NoError(t, err)

		// Verify expectations
		mockSystemdManager.AssertExpectations(t)
		mockCaddyManager.AssertExpectations(t)
	})

	// Test service deletion
	t.Run("ServiceDeletion", func(t *testing.T) {
		// Create test service
		service := &models.Service{
			ID:        "test123",
			Name:      "testservice4",
			Port:      8095,
			Version:   "0.20.0",
			Subdomain: "test4",
			Status:    "running",
		}

		// Set up expectations
		mockSystemdManager.On("RemoveService", "testservice4").Return(nil).Once()
		mockCaddyManager.On("RemoveConfiguration", "testservice4").Return(nil).Once()

		// Delete service
		err := scm.DeleteService(service)
		assert.NoError(t, err)

		// Verify expectations
		mockSystemdManager.AssertExpectations(t)
		mockCaddyManager.AssertExpectations(t)
	})
}
