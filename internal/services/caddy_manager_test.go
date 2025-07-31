package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tigawanna/pockestrator/internal/models"
)

// TestCaddyManager tests the CaddyManager implementation
func TestCaddyManager(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "caddy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary Caddyfile
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")

	// Create test domain
	testDomain := "example.com"

	// Create a mock CaddyManager that overrides the ReloadCaddy method
	cm := &MockCaddyManager{
		CaddyManagerImpl: CaddyManagerImpl{
			caddyfilePath: caddyfilePath,
			domain:        testDomain,
		},
	}

	// Create test service
	service := &models.Service{
		ID:        "test123",
		Name:      "testservice",
		Port:      8091,
		Version:   "0.20.0",
		Subdomain: "testservice",
		Status:    "running",
	}

	// Test adding configuration
	t.Run("AddConfiguration", func(t *testing.T) {
		if err := cm.AddConfiguration(service); err != nil {
			t.Fatalf("Failed to add configuration: %v", err)
		}

		// Verify configuration was added
		content, err := os.ReadFile(caddyfilePath)
		if err != nil {
			t.Fatalf("Failed to read Caddyfile: %v", err)
		}

		expectedConfig := `testservice.example.com {
    reverse_proxy 127.0.0.1:8091
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

`
		if string(content) != expectedConfig {
			t.Errorf("Configuration doesn't match expected.\nGot:\n%s\nExpected:\n%s", string(content), expectedConfig)
		}

		// Test validation
		if !cm.ValidateConfiguration(service) {
			t.Errorf("Configuration validation failed when it should succeed")
		}
	})

	// Test updating existing configuration
	t.Run("UpdateConfiguration", func(t *testing.T) {
		// Update service port
		updatedService := &models.Service{
			ID:        "test123",
			Name:      "testservice",
			Port:      8092, // Changed port
			Version:   "0.20.0",
			Subdomain: "testservice",
			Status:    "running",
		}

		if err := cm.AddConfiguration(updatedService); err != nil {
			t.Fatalf("Failed to update configuration: %v", err)
		}

		// Verify configuration was updated
		content, err := os.ReadFile(caddyfilePath)
		if err != nil {
			t.Fatalf("Failed to read Caddyfile: %v", err)
		}

		expectedConfig := `testservice.example.com {
    reverse_proxy 127.0.0.1:8092
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

`
		if string(content) != expectedConfig {
			t.Errorf("Updated configuration doesn't match expected.\nGot:\n%s\nExpected:\n%s", string(content), expectedConfig)
		}

		// Test validation with old service should fail
		if cm.ValidateConfiguration(service) {
			t.Errorf("Configuration validation succeeded when it should fail for old service")
		}

		// Test validation with updated service should succeed
		if !cm.ValidateConfiguration(updatedService) {
			t.Errorf("Configuration validation failed when it should succeed for updated service")
		}
	})

	// Test removing configuration
	t.Run("RemoveConfiguration", func(t *testing.T) {
		if err := cm.RemoveConfiguration(service.Name); err != nil {
			t.Fatalf("Failed to remove configuration: %v", err)
		}

		// Verify configuration was removed
		content, err := os.ReadFile(caddyfilePath)
		if err != nil {
			t.Fatalf("Failed to read Caddyfile: %v", err)
		}

		if string(content) != "" {
			t.Errorf("Configuration was not removed properly. Got:\n%s", string(content))
		}

		// Test validation after removal
		if cm.ValidateConfiguration(service) {
			t.Errorf("Configuration validation succeeded when it should fail after removal")
		}
	})

	// Test multiple services
	t.Run("MultipleServices", func(t *testing.T) {
		// Create two services
		service1 := &models.Service{
			ID:        "test1",
			Name:      "service1",
			Port:      8091,
			Version:   "0.20.0",
			Subdomain: "service1",
			Status:    "running",
		}

		service2 := &models.Service{
			ID:        "test2",
			Name:      "service2",
			Port:      8092,
			Version:   "0.20.0",
			Subdomain: "service2",
			Status:    "running",
		}

		// Add both services
		if err := cm.AddConfiguration(service1); err != nil {
			t.Fatalf("Failed to add service1: %v", err)
		}

		if err := cm.AddConfiguration(service2); err != nil {
			t.Fatalf("Failed to add service2: %v", err)
		}

		// Verify both configurations exist
		content, err := os.ReadFile(caddyfilePath)
		if err != nil {
			t.Fatalf("Failed to read Caddyfile: %v", err)
		}

		expectedConfig := `service1.example.com {
    reverse_proxy 127.0.0.1:8091
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

service2.example.com {
    reverse_proxy 127.0.0.1:8092
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

`
		if string(content) != expectedConfig {
			t.Errorf("Multiple configurations don't match expected.\nGot:\n%s\nExpected:\n%s", string(content), expectedConfig)
		}

		// Remove one service
		if err := cm.RemoveConfiguration(service1.Name); err != nil {
			t.Fatalf("Failed to remove service1: %v", err)
		}

		// Verify only service2 remains
		content, err = os.ReadFile(caddyfilePath)
		if err != nil {
			t.Fatalf("Failed to read Caddyfile: %v", err)
		}

		expectedConfig = `service2.example.com {
    reverse_proxy 127.0.0.1:8092
    header Strict-Transport-Security max-age=31536000;
    encode gzip
}

`
		if string(content) != expectedConfig {
			t.Errorf("After removal, configuration doesn't match expected.\nGot:\n%s\nExpected:\n%s", string(content), expectedConfig)
		}
	})
}

// MockCaddyManager is a test implementation of CaddyManager that overrides ReloadCaddy
type MockCaddyManager struct {
	CaddyManagerImpl
}

// ReloadCaddy is a mock implementation that does nothing
func (m *MockCaddyManager) ReloadCaddy() error {
	return nil
}
