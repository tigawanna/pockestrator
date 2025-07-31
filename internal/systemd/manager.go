package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// ServiceTemplate is the systemd service file template
const ServiceTemplate = `[Unit]
Description = {{.ProjectName}} pocketbase

[Service]
Type           = simple
User           = root
Group          = root
LimitNOFILE    = 4096
Restart        = always
RestartSec     = 5s
StandardOutput   = append:{{.ServiceDir}}/errors.log
StandardError    = append:{{.ServiceDir}}/errors.log
WorkingDirectory = {{.ServiceDir}}/
ExecStart      = {{.ServiceDir}}/pocketbase serve --http="127.0.0.1:{{.Port}}"

[Install]
WantedBy = multi-user.target
`

// Manager handles systemd service operations
type Manager struct {
	systemdDir string
}

// ServiceConfig holds the configuration for generating systemd service files
type ServiceConfig struct {
	ProjectName string
	ServiceDir  string
	Port        int
}

// NewManager creates a new systemd manager
func NewManager(systemdDir string) *Manager {
	return &Manager{
		systemdDir: systemdDir,
	}
}

// CreateService creates a systemd service file
func (m *Manager) CreateService(config *ServiceConfig) error {
	// Parse template
	tmpl, err := template.New("service").Parse(ServiceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	// Create service file path
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", config.ProjectName)
	serviceFilePath := filepath.Join(m.systemdDir, serviceFileName)

	// Create service file
	file, err := os.Create(serviceFilePath)
	if err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(serviceFilePath, 0644); err != nil {
		return fmt.Errorf("failed to set service file permissions: %w", err)
	}

	return nil
}

// EnableService enables and starts a systemd service
func (m *Manager) EnableService(serviceName string) error {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	// Reload systemd daemon
	if err := m.reloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable service
	cmd := exec.Command("sudo", "systemctl", "enable", serviceFileName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Start service
	cmd = exec.Command("sudo", "systemctl", "start", serviceFileName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// DisableService stops and disables a systemd service
func (m *Manager) DisableService(serviceName string) error {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	// Stop service
	cmd := exec.Command("sudo", "systemctl", "stop", serviceFileName)
	cmd.Run() // Don't fail if already stopped

	// Disable service
	cmd = exec.Command("sudo", "systemctl", "disable", serviceFileName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable service: %w", err)
	}

	return nil
}

// RemoveService removes a systemd service file
func (m *Manager) RemoveService(serviceName string) error {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)
	serviceFilePath := filepath.Join(m.systemdDir, serviceFileName)

	// First disable the service
	if err := m.DisableService(serviceName); err != nil {
		// Log but don't fail
	}

	// Remove service file
	if err := os.Remove(serviceFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload daemon
	return m.reloadDaemon()
}

// GetServiceStatus returns the status of a systemd service
func (m *Manager) GetServiceStatus(serviceName string) (string, error) {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("sudo", "systemctl", "is-active", serviceFileName)
	output, err := cmd.Output()
	if err != nil {
		return "inactive", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// IsServiceEnabled checks if a service is enabled
func (m *Manager) IsServiceEnabled(serviceName string) (bool, error) {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("sudo", "systemctl", "is-enabled", serviceFileName)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	status := strings.TrimSpace(string(output))
	return status == "enabled", nil
}

// GetServiceLogs returns the last n lines of service logs
func (m *Manager) GetServiceLogs(serviceName string, lines int) (string, error) {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("sudo", "journalctl", "-u", serviceFileName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get service logs: %w", err)
	}

	return string(output), nil
}

// RestartService restarts a systemd service
func (m *Manager) RestartService(serviceName string) error {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)

	cmd := exec.Command("sudo", "systemctl", "restart", serviceFileName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	return nil
}

// reloadDaemon reloads the systemd daemon
func (m *Manager) reloadDaemon() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	return nil
}

// ValidateServiceFile checks if a service file exists and is valid
func (m *Manager) ValidateServiceFile(serviceName string) error {
	serviceFileName := fmt.Sprintf("%s-pocketbase.service", serviceName)
	serviceFilePath := filepath.Join(m.systemdDir, serviceFileName)

	// Check if file exists
	if _, err := os.Stat(serviceFilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service file does not exist: %s", serviceFilePath)
		}
		return fmt.Errorf("failed to check service file: %w", err)
	}

	// Basic validation - check if systemd can parse it
	cmd := exec.Command("sudo", "systemd-analyze", "verify", serviceFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("service file validation failed: %w", err)
	}

	return nil
}
