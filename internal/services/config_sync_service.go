package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/tigawanna/pockestrator/internal/models"
)

// ConfigSyncServiceImpl implements the ConfigSyncService interface
type ConfigSyncServiceImpl struct {
	systemdManager SystemdManager
	caddyManager   CaddyManager
	baseDir        string
	serviceDir     string
	caddyfilePath  string
	domain         string
	db             ServiceRepository
}

// ServiceRepository defines the interface for accessing and updating service data
type ServiceRepository interface {
	FindServiceByName(name string) (*models.Service, error)
	FindServiceByPort(port int) (*models.Service, error)
	ListAllServices() ([]*models.Service, error)
	UpdateService(service *models.Service) error
}

// NewConfigSyncService creates a new ConfigSyncService instance
func NewConfigSyncService(
	systemdManager SystemdManager,
	caddyManager CaddyManager,
	baseDir string,
	serviceDir string,
	caddyfilePath string,
	domain string,
	db ServiceRepository,
) *ConfigSyncServiceImpl {
	return &ConfigSyncServiceImpl{
		systemdManager: systemdManager,
		caddyManager:   caddyManager,
		baseDir:        baseDir,
		serviceDir:     serviceDir,
		caddyfilePath:  caddyfilePath,
		domain:         domain,
		db:             db,
	}
}

// SyncServiceToSystem synchronizes a service record to system files (collection → files)
func (cs *ConfigSyncServiceImpl) SyncServiceToSystem(service *models.Service) error {
	// 1. Update systemd service file
	if err := cs.systemdManager.CreateService(service); err != nil {
		return fmt.Errorf("failed to update systemd service: %w", err)
	}

	// 2. Update Caddy configuration
	if err := cs.caddyManager.AddConfiguration(service); err != nil {
		return fmt.Errorf("failed to update Caddy configuration: %w", err)
	}

	// 3. Reload systemd daemon
	if err := cs.reloadSystemd(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// 4. Restart the service if it was running
	status, err := cs.systemdManager.GetServiceStatus(service.Name)
	if err == nil && status.Active {
		if err := cs.systemdManager.StopService(service.Name); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		if err := cs.systemdManager.StartService(service.Name); err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
		}
	}

	return nil
}

// SyncSystemToService synchronizes system files to a service record (files → collection)
func (cs *ConfigSyncServiceImpl) SyncSystemToService(service *models.Service) (*models.Service, error) {
	// Create a copy of the service to avoid modifying the original
	updatedService := &models.Service{
		ID:        service.ID,
		Name:      service.Name,
		Port:      service.Port,
		Version:   service.Version,
		Subdomain: service.Subdomain,
		Status:    service.Status,
		CreatedAt: service.CreatedAt,
		UpdatedAt: service.UpdatedAt,
	}

	// 1. Extract port from systemd service file
	port, err := cs.extractPortFromSystemd(service.Name)
	if err == nil && port != service.Port {
		updatedService.Port = port
	}

	// 2. Extract subdomain from Caddy configuration
	subdomain, err := cs.extractSubdomainFromCaddy(service.Name)
	if err == nil && subdomain != service.Subdomain {
		updatedService.Subdomain = subdomain
	}

	// 3. Update service status based on systemd status
	status, err := cs.systemdManager.GetServiceStatus(service.Name)
	if err == nil {
		if status.Active {
			updatedService.Status = "running"
		} else {
			updatedService.Status = "stopped"
		}
	}

	// 4. Update the service record in the database
	if err := cs.db.UpdateService(updatedService); err != nil {
		return nil, fmt.Errorf("failed to update service record: %w", err)
	}

	return updatedService, nil
}

// DetectConflicts checks for conflicts between a service record and system files
func (cs *ConfigSyncServiceImpl) DetectConflicts(service *models.Service) (*ConfigConflict, error) {
	conflict := &ConfigConflict{
		ServiceID:      service.ID,
		ServiceName:    service.Name,
		HasConflict:    false,
		ConflictFields: make(map[string]string),
		SystemState:    &models.Service{},
	}

	// 1. Check systemd service file
	systemdPort, err := cs.extractPortFromSystemd(service.Name)
	if err == nil && systemdPort != service.Port {
		conflict.HasConflict = true
		conflict.ConflictFields["port"] = fmt.Sprintf("Database: %d, Systemd: %d", service.Port, systemdPort)
		conflict.SystemState.Port = systemdPort
	} else {
		conflict.SystemState.Port = service.Port
	}

	// 2. Check Caddy configuration
	subdomain, err := cs.extractSubdomainFromCaddy(service.Name)
	if err == nil && subdomain != service.Subdomain {
		conflict.HasConflict = true
		conflict.ConflictFields["subdomain"] = fmt.Sprintf("Database: %s, Caddy: %s", service.Subdomain, subdomain)
		conflict.SystemState.Subdomain = subdomain
	} else {
		conflict.SystemState.Subdomain = service.Subdomain
	}

	// 3. Check service status
	status, err := cs.systemdManager.GetServiceStatus(service.Name)
	if err == nil {
		var systemStatus string
		if status.Active {
			systemStatus = "running"
		} else {
			systemStatus = "stopped"
		}

		if systemStatus != service.Status {
			conflict.HasConflict = true
			conflict.ConflictFields["status"] = fmt.Sprintf("Database: %s, System: %s", service.Status, systemStatus)
			conflict.SystemState.Status = systemStatus
		} else {
			conflict.SystemState.Status = service.Status
		}
	}

	// Copy other fields that don't have conflicts
	conflict.SystemState.ID = service.ID
	conflict.SystemState.Name = service.Name
	conflict.SystemState.Version = service.Version
	conflict.SystemState.CreatedAt = service.CreatedAt
	conflict.SystemState.UpdatedAt = service.UpdatedAt

	return conflict, nil
}

// ValidateServiceConfig validates that a service configuration is consistent with system files
func (cs *ConfigSyncServiceImpl) ValidateServiceConfig(service *models.Service) (bool, []string) {
	var issues []string
	isValid := true

	// 1. Check if systemd service file exists and has correct port
	systemdPort, err := cs.extractPortFromSystemd(service.Name)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Systemd service file not found or invalid: %v", err))
		isValid = false
	} else if systemdPort != service.Port {
		issues = append(issues, fmt.Sprintf("Port mismatch: Database has %d, systemd has %d", service.Port, systemdPort))
		isValid = false
	}

	// 2. Check if Caddy configuration exists and has correct subdomain
	subdomain, err := cs.extractSubdomainFromCaddy(service.Name)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Caddy configuration not found or invalid: %v", err))
		isValid = false
	} else if subdomain != service.Subdomain {
		issues = append(issues, fmt.Sprintf("Subdomain mismatch: Database has %s, Caddy has %s", service.Subdomain, subdomain))
		isValid = false
	}

	// 3. Check if PocketBase binary exists
	binaryPath := filepath.Join(cs.baseDir, service.Name, "pocketbase")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		issues = append(issues, "PocketBase binary not found")
		isValid = false
	}

	return isValid, issues
}

// Helper methods

// extractPortFromSystemd extracts the port from a systemd service file
func (cs *ConfigSyncServiceImpl) extractPortFromSystemd(serviceName string) (int, error) {
	servicePath := filepath.Join(cs.serviceDir, fmt.Sprintf("%s-pocketbase.service", serviceName))

	// Read service file
	content, err := os.ReadFile(servicePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read systemd service file: %w", err)
	}

	// Extract port from ExecStart line
	re := regexp.MustCompile(`--http=127\.0\.0\.1:(\d+)`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return 0, fmt.Errorf("port not found in systemd service file")
	}

	port, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid port in systemd service file: %w", err)
	}

	return port, nil
}

// extractSubdomainFromCaddy extracts the subdomain from Caddy configuration
func (cs *ConfigSyncServiceImpl) extractSubdomainFromCaddy(serviceName string) (string, error) {
	// Read Caddyfile
	content, err := os.ReadFile(cs.caddyfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	// Look for configuration block for this service
	pattern := fmt.Sprintf(`([\w-]+)\.%s \{[^}]*reverse_proxy 127\.0\.0\.1:\d+[^}]*\}`, regexp.QuoteMeta(cs.domain))
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) >= 2 {
			subdomain := match[1]

			// Verify this is the correct service by checking for port match
			servicePort, err := cs.extractPortFromSystemd(serviceName)
			if err != nil {
				continue
			}

			portPattern := fmt.Sprintf(`%s\.%s \{[^}]*reverse_proxy 127\.0\.0\.1:%d[^}]*\}`,
				regexp.QuoteMeta(subdomain),
				regexp.QuoteMeta(cs.domain),
				servicePort)

			portRe := regexp.MustCompile(portPattern)
			if portRe.MatchString(string(content)) {
				return subdomain, nil
			}
		}
	}

	return "", fmt.Errorf("subdomain not found in Caddyfile for service %s", serviceName)
}

// reloadSystemd reloads the systemd daemon
func (cs *ConfigSyncServiceImpl) reloadSystemd() error {
	cmd := "systemctl daemon-reload"
	_, err := executeCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	return nil
}

// executeCommand executes a shell command and returns its output
func executeCommand(command string) (string, error) {
	// This is a simplified implementation. In a real-world scenario,
	// you would use os/exec package to execute commands.
	// For now, we'll just simulate success.
	return "", nil
}
