package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Manager handles PocketBase service management
type Manager struct {
	baseDir     string
	systemdDir  string
	caddyConfig string
}

// NewManager creates a new service manager
func NewManager(baseDir, systemdDir, caddyConfig string) *Manager {
	return &Manager{
		baseDir:     baseDir,
		systemdDir:  systemdDir,
		caddyConfig: caddyConfig,
	}
}

// Deploy deploys a new PocketBase service
func (m *Manager) Deploy(ctx context.Context, config *DeploymentConfig) error {
	// Create service directory
	serviceDir := filepath.Join(m.baseDir, config.ProjectName)
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Download and extract PocketBase
	if err := m.downloadPocketBase(ctx, config.PocketBaseVersion, serviceDir); err != nil {
		return fmt.Errorf("failed to download PocketBase: %w", err)
	}

	// Set executable permissions
	pbPath := filepath.Join(serviceDir, "pocketbase")
	if err := os.Chmod(pbPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// downloadPocketBase downloads and extracts the specified PocketBase version
func (m *Manager) downloadPocketBase(ctx context.Context, version, destDir string) error {
	// Construct download URL
	url := fmt.Sprintf("https://github.com/pocketbase/pocketbase/releases/download/v%s/pocketbase_%s_linux_amd64.zip", version, version)

	// Create temporary file
	tempFile, err := os.CreateTemp("", "pocketbase_*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Download the file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download PocketBase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download PocketBase: status %d", resp.StatusCode)
	}

	// Write to temp file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write download: %w", err)
	}

	// Extract zip file
	return m.extractZip(tempFile.Name(), destDir)
}

// extractZip extracts a zip file to the destination directory
func (m *Manager) extractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)

		// Security check
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			fileReader.Close()
			return fmt.Errorf("failed to create target file: %w", err)
		}

		_, err = io.Copy(targetFile, fileReader)
		fileReader.Close()
		targetFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

// GetServiceStatus checks if a service is running
func (m *Manager) GetServiceStatus(serviceName string) (*HealthStatus, error) {
	status := &HealthStatus{
		ServiceID:   serviceName,
		LastChecked: time.Now(),
	}

	// Check systemd status
	cmd := exec.Command("sudo", "systemctl", "is-active", serviceName+"-pocketbase.service")
	output, err := cmd.Output()
	if err != nil {
		status.SystemdStatus = "inactive"
		status.IsRunning = false
		status.ErrorMessage = err.Error()
	} else {
		status.SystemdStatus = strings.TrimSpace(string(output))
		status.IsRunning = status.SystemdStatus == "active"
	}

	return status, nil
}

// Stop stops a PocketBase service
func (m *Manager) Stop(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "stop", serviceName+"-pocketbase.service")
	return cmd.Run()
}

// Start starts a PocketBase service
func (m *Manager) Start(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "start", serviceName+"-pocketbase.service")
	return cmd.Run()
}

// Restart restarts a PocketBase service
func (m *Manager) Restart(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "restart", serviceName+"-pocketbase.service")
	return cmd.Run()
}

// Remove removes a PocketBase service completely
func (m *Manager) Remove(ctx context.Context, config *ServiceConfig) error {
	// Stop the service
	if err := m.Stop(config.ProjectName); err != nil {
		// Log error but continue with removal
	}

	// Disable the service
	cmd := exec.Command("sudo", "systemctl", "disable", config.ProjectName+"-pocketbase.service")
	cmd.Run()

	// Remove systemd service file
	serviceFile := filepath.Join(m.systemdDir, config.ProjectName+"-pocketbase.service")
	os.Remove(serviceFile)

	// Reload systemd daemon
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	cmd.Run()

	// Remove service directory
	serviceDir := filepath.Join(m.baseDir, config.ProjectName)
	if err := os.RemoveAll(serviceDir); err != nil {
		return fmt.Errorf("failed to remove service directory: %w", err)
	}

	return nil
}

// GetLatestVersion fetches the latest PocketBase version from GitHub
func (m *Manager) GetLatestVersion(ctx context.Context) (string, error) {
	// For now, return a default version. In production, this would query GitHub API
	return "0.28.4", nil
}

// GetNextPort finds the next available port starting from 8091
func (m *Manager) GetNextPort(usedPorts []int) int {
	basePort := 8091

	// Create map for O(1) lookups
	portMap := make(map[int]bool)
	for _, port := range usedPorts {
		portMap[port] = true
	}

	// Find next available port
	for port := basePort; port <= 65535; port++ {
		if !portMap[port] {
			return port
		}
	}

	return basePort // fallback
}

// IsPortAvailable checks if a port is available on the system
func (m *Manager) IsPortAvailable(port int) bool {
	// Use netstat or similar to check port availability
	cmd := exec.Command("netstat", "-tuln")
	output, err := cmd.Output()
	if err != nil {
		return true // assume available if can't check
	}

	portStr := ":" + strconv.Itoa(port)
	return !strings.Contains(string(output), portStr)
}
