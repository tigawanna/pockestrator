package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tigawanna/pockestrator/internal/models"
)

func TestNewSystemdManager(t *testing.T) {
	manager := NewSystemdManager()

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.serviceDir != "/lib/systemd/system" {
		t.Errorf("Expected serviceDir to be /lib/systemd/system, got %s", manager.serviceDir)
	}
}

func TestNewSystemdManagerWithDir(t *testing.T) {
	customDir := "/tmp/test-systemd"
	manager := NewSystemdManagerWithDir(customDir)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	if manager.serviceDir != customDir {
		t.Errorf("Expected serviceDir to be %s, got %s", customDir, manager.serviceDir)
	}
}

func TestCreateService(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "systemd_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewSystemdManagerWithDir(tempDir)

	service := &models.Service{
		Name: "test-service",
		Port: 8091,
	}

	t.Run("create service file", func(t *testing.T) {
		err := manager.CreateService(service)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check if service file was created
		servicePath := filepath.Join(tempDir, "test-service-pocketbase.service")
		if _, err := os.Stat(servicePath); os.IsNotExist(err) {
			t.Error("Expected service file to be created")
		}

		// Check service file content
		content, err := os.ReadFile(servicePath)
		if err != nil {
			t.Fatalf("Failed to read service file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Description=PocketBase service for test-service") {
			t.Error("Service file should contain correct description")
		}

		if !strings.Contains(contentStr, "ExecStart=/home/ubuntu/test-service/pocketbase serve --http=127.0.0.1:8091") {
			t.Error("Service file should contain correct ExecStart command")
		}

		if !strings.Contains(contentStr, "WorkingDirectory=/home/ubuntu/test-service") {
			t.Error("Service file should contain correct WorkingDirectory")
		}

		if !strings.Contains(contentStr, "User=ubuntu") {
			t.Error("Service file should contain correct User")
		}

		if !strings.Contains(contentStr, "Restart=always") {
			t.Error("Service file should contain restart policy")
		}
	})
}

func TestGenerateServiceContent(t *testing.T) {
	manager := NewSystemdManager()

	service := &models.Service{
		Name: "my-app",
		Port: 8092,
	}

	content := manager.generateServiceContent(service)

	t.Run("contains required sections", func(t *testing.T) {
		if !strings.Contains(content, "[Unit]") {
			t.Error("Service content should contain [Unit] section")
		}

		if !strings.Contains(content, "[Service]") {
			t.Error("Service content should contain [Service] section")
		}

		if !strings.Contains(content, "[Install]") {
			t.Error("Service content should contain [Install] section")
		}
	})

	t.Run("contains correct configuration", func(t *testing.T) {
		if !strings.Contains(content, "Description=PocketBase service for my-app") {
			t.Error("Service content should contain correct description")
		}

		if !strings.Contains(content, "ExecStart=/home/ubuntu/my-app/pocketbase serve --http=127.0.0.1:8092") {
			t.Error("Service content should contain correct ExecStart")
		}

		if !strings.Contains(content, "WorkingDirectory=/home/ubuntu/my-app") {
			t.Error("Service content should contain correct WorkingDirectory")
		}

		if !strings.Contains(content, "User=ubuntu") {
			t.Error("Service content should contain correct User")
		}

		if !strings.Contains(content, "Group=ubuntu") {
			t.Error("Service content should contain correct Group")
		}

		if !strings.Contains(content, "Type=simple") {
			t.Error("Service content should contain correct Type")
		}

		if !strings.Contains(content, "Restart=always") {
			t.Error("Service content should contain restart policy")
		}

		if !strings.Contains(content, "RestartSec=5") {
			t.Error("Service content should contain restart delay")
		}

		if !strings.Contains(content, "StandardOutput=append:/home/ubuntu/my-app/service.log") {
			t.Error("Service content should contain correct StandardOutput")
		}

		if !strings.Contains(content, "StandardError=append:/home/ubuntu/my-app/service.log") {
			t.Error("Service content should contain correct StandardError")
		}

		if !strings.Contains(content, "WantedBy=multi-user.target") {
			t.Error("Service content should contain correct WantedBy")
		}
	})
}

// Note: The following tests would require mocking systemctl commands in a real environment
// For now, we'll test the basic error handling when systemctl is not available

func TestEnableService(t *testing.T) {
	manager := NewSystemdManager()

	t.Run("enable service with missing systemctl", func(t *testing.T) {
		// This test will fail in environments without systemctl, which is expected
		err := manager.EnableService("test-service")
		if err == nil {
			// If no error, systemctl is available and service was enabled
			t.Log("systemctl is available, service enable attempted")
		} else {
			// Expected in test environments without systemctl
			if !strings.Contains(err.Error(), "failed to enable service") {
				t.Errorf("Expected enable service error, got: %v", err)
			}
		}
	})
}

func TestStartService(t *testing.T) {
	manager := NewSystemdManager()

	t.Run("start service with missing systemctl", func(t *testing.T) {
		// This test will fail in environments without systemctl, which is expected
		err := manager.StartService("test-service")
		if err == nil {
			// If no error, systemctl is available and service was started
			t.Log("systemctl is available, service start attempted")
		} else {
			// Expected in test environments without systemctl
			if !strings.Contains(err.Error(), "failed to start service") {
				t.Errorf("Expected start service error, got: %v", err)
			}
		}
	})
}

func TestStopService(t *testing.T) {
	manager := NewSystemdManager()

	t.Run("stop service with missing systemctl", func(t *testing.T) {
		// This test will fail in environments without systemctl, which is expected
		err := manager.StopService("test-service")
		if err == nil {
			// If no error, systemctl is available and service was stopped
			t.Log("systemctl is available, service stop attempted")
		} else {
			// Expected in test environments without systemctl
			if !strings.Contains(err.Error(), "failed to stop service") {
				t.Errorf("Expected stop service error, got: %v", err)
			}
		}
	})
}

func TestIsServiceRunning(t *testing.T) {
	manager := NewSystemdManager()

	t.Run("check if service is running", func(t *testing.T) {
		// This will return false in environments without systemctl or when service doesn't exist
		running := manager.IsServiceRunning("test-service")

		// In test environments, this should typically be false
		if running {
			t.Log("Service appears to be running (systemctl available)")
		} else {
			t.Log("Service is not running or systemctl not available (expected in test)")
		}
	})
}

func TestGetServiceStatus(t *testing.T) {
	manager := NewSystemdManager()

	t.Run("get service status", func(t *testing.T) {
		status, err := manager.GetServiceStatus("test-service")
		if err != nil {
			t.Errorf("Expected no error from GetServiceStatus, got: %v", err)
		}

		if status == nil {
			t.Fatal("Expected status to be returned, got nil")
		}

		if status.Name != "test-service" {
			t.Errorf("Expected status name to be test-service, got %s", status.Name)
		}

		// In test environments without systemctl, service should not be active
		if status.Active {
			t.Log("Service appears to be active (systemctl available)")
		} else {
			t.Log("Service is not active (expected in test environment)")
		}
	})
}

func TestRemoveService(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "systemd_remove_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewSystemdManagerWithDir(tempDir)

	// Create a test service file
	servicePath := filepath.Join(tempDir, "test-service-pocketbase.service")
	if err := os.WriteFile(servicePath, []byte("test service content"), 0644); err != nil {
		t.Fatalf("Failed to create test service file: %v", err)
	}

	t.Run("remove service file", func(t *testing.T) {
		_ = manager.RemoveService("test-service")

		// The function will try to run systemctl commands which will fail in test environment
		// but it should still remove the service file
		if _, err := os.Stat(servicePath); !os.IsNotExist(err) {
			t.Error("Expected service file to be removed")
		}
	})
}
