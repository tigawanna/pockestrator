package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tigawanna/pockestrator/internal/models"
)

// SystemdManagerImpl implements the SystemdManager interface
type SystemdManagerImpl struct {
	serviceDir string
}

// NewSystemdManager creates a new systemd manager instance
func NewSystemdManager() *SystemdManagerImpl {
	return &SystemdManagerImpl{
		serviceDir: "/lib/systemd/system",
	}
}

// NewSystemdManagerWithDir creates a new systemd manager with custom service directory
func NewSystemdManagerWithDir(serviceDir string) *SystemdManagerImpl {
	return &SystemdManagerImpl{
		serviceDir: serviceDir,
	}
}

// CreateService creates a systemd service file for the given service
func (sm *SystemdManagerImpl) CreateService(service *models.Service) error {
	serviceName := fmt.Sprintf("%s-pocketbase.service", service.Name)
	servicePath := filepath.Join(sm.serviceDir, serviceName)

	// Generate service file content
	serviceContent := sm.generateServiceContent(service)

	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	return nil
}

// EnableService enables a systemd service
func (sm *SystemdManagerImpl) EnableService(serviceName string) error {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("systemctl", "enable", fullServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable service %s: %w, output: %s", fullServiceName, err, string(output))
	}

	return nil
}

// StartService starts a systemd service
func (sm *SystemdManagerImpl) StartService(serviceName string) error {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("systemctl", "start", fullServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w, output: %s", fullServiceName, err, string(output))
	}

	return nil
}

// StopService stops a systemd service
func (sm *SystemdManagerImpl) StopService(serviceName string) error {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("systemctl", "stop", fullServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w, output: %s", fullServiceName, err, string(output))
	}

	return nil
}

// RemoveService removes a systemd service
func (sm *SystemdManagerImpl) RemoveService(serviceName string) error {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	// Stop the service first
	if err := sm.StopService(serviceName); err != nil {
		// Continue even if stop fails (service might not be running)
	}

	// Disable the service
	cmd := exec.Command("systemctl", "disable", fullServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Continue even if disable fails (service might not be enabled)
		fmt.Printf("Warning: failed to disable service %s: %v, output: %s\n", fullServiceName, err, string(output))
	}

	// Remove service file
	servicePath := filepath.Join(sm.serviceDir, fullServiceName)
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd daemon
	cmd = exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w, output: %s", err, string(output))
	}

	return nil
}

// IsServiceRunning checks if a systemd service is currently running
func (sm *SystemdManagerImpl) IsServiceRunning(serviceName string) bool {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("systemctl", "is-active", fullServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "active"
}

// GetServiceStatus gets the detailed status of a systemd service
func (sm *SystemdManagerImpl) GetServiceStatus(serviceName string) (*ServiceStatus, error) {
	fullServiceName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	status := &ServiceStatus{
		Name: serviceName,
	}

	// Check if service is active
	cmd := exec.Command("systemctl", "is-active", fullServiceName)
	output, err := cmd.CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		status.Active = true
		status.Status = "active"
	} else {
		status.Active = false
		status.Status = strings.TrimSpace(string(output))
	}

	// Check if service is enabled
	cmd = exec.Command("systemctl", "is-enabled", fullServiceName)
	output, err = cmd.CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) == "enabled" {
		status.Enabled = true
	} else {
		status.Enabled = false
	}

	return status, nil
}

// generateServiceContent generates the systemd service file content
func (sm *SystemdManagerImpl) generateServiceContent(service *models.Service) string {
	workingDir := fmt.Sprintf("/home/ubuntu/%s", service.Name)
	execStart := fmt.Sprintf("/home/ubuntu/%s/pocketbase serve --http=127.0.0.1:%d", service.Name, service.Port)

	return fmt.Sprintf(`[Unit]
Description=PocketBase service for %s
After=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=%s
ExecStart=%s
Restart=always
RestartSec=5
StandardOutput=append:/home/ubuntu/%s/service.log
StandardError=append:/home/ubuntu/%s/service.log

[Install]
WantedBy=multi-user.target
`, service.Name, workingDir, execStart, service.Name, service.Name)
}
