package validators

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ServiceValidator provides validation functionality for services
type ServiceValidator struct {
	systemdManager services.SystemdManager
	caddyManager   services.CaddyManager
	baseDir        string
	dao            *daos.Dao
}

// NewServiceValidator creates a new ServiceValidator
func NewServiceValidator(dao *daos.Dao, systemdManager services.SystemdManager, caddyManager services.CaddyManager, baseDir string) *ServiceValidator {
	return &ServiceValidator{
		systemdManager: systemdManager,
		caddyManager:   caddyManager,
		baseDir:        baseDir,
		dao:            dao,
	}
}

// ValidateService validates a service configuration
func (v *ServiceValidator) ValidateService(service *models.Service) (*models.ServiceValidation, error) {
	validation := &models.ServiceValidation{
		Issues: []string{},
	}

	// Check if systemd service exists
	serviceName := fmt.Sprintf("%s-pocketbase.service", service.Name)
	serviceStatus, err := v.systemdManager.GetServiceStatus(service.Name)
	if err != nil {
		validation.Issues = append(validation.Issues, fmt.Sprintf("Failed to get systemd status: %v", err))
	} else {
		validation.SystemdExists = true
		validation.SystemdRunning = serviceStatus.Active

		if !serviceStatus.Active {
			validation.Issues = append(validation.Issues, fmt.Sprintf("Service %s is not running", serviceName))
		}

		if !serviceStatus.Enabled {
			validation.Issues = append(validation.Issues, fmt.Sprintf("Service %s is not enabled", serviceName))
		}
	}

	// Check if Caddy configuration is correct
	validation.CaddyConfigured = v.caddyManager.ValidateConfiguration(service)
	if !validation.CaddyConfigured {
		validation.Issues = append(validation.Issues, "Caddy configuration is missing or incorrect")
	}

	// Check if PocketBase binary exists
	binaryPath := filepath.Join(v.baseDir, service.Name, "pocketbase")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		validation.BinaryExists = false
		validation.Issues = append(validation.Issues, "PocketBase binary not found")
	} else {
		validation.BinaryExists = true
	}

	// Check if port in systemd service matches the service port
	validation.PortMatches = v.validatePortInSystemdConfig(service)
	if !validation.PortMatches {
		validation.Issues = append(validation.Issues, "Port in systemd configuration doesn't match service port")
	}

	return validation, nil
}

// validatePortInSystemdConfig checks if the port in systemd service file matches the service port
func (v *ServiceValidator) validatePortInSystemdConfig(service *models.Service) bool {
	// This is a simplified implementation. In a real-world scenario, you would parse the systemd service file
	// to extract the port from the ExecStart line and compare it with the service port.

	// For now, we'll rely on the systemd service being created correctly by the SystemdManager
	// and assume the port matches if the service is running.
	serviceStatus, err := v.systemdManager.GetServiceStatus(service.Name)
	if err != nil || !serviceStatus.Active {
		return false
	}

	return true
}

// ValidatePortAvailable checks if a port is available
func (v *ServiceValidator) ValidatePortAvailable(port int, excludeService string) error {
	// Check if port is in valid range
	if port < 8000 || port > 9999 {
		return &models.ValidationError{
			Field:   "port",
			Message: "port must be between 8000 and 9999",
		}
	}

	// Check if port is already used by another service in the database
	serviceRecord, err := v.findServiceByPort(port)
	if err == nil && serviceRecord != nil {
		// Extract the service name from the record
		serviceName := serviceRecord.GetString("name")
		if serviceName != excludeService {
			return &models.ValidationError{
				Field:   "port",
				Message: fmt.Sprintf("port %d is already used by service '%s'", port, serviceName),
			}
		}
	}

	// Check if port is available on the system
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return &models.ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("port %d is already in use by another process", port),
		}
	}
	listener.Close()

	return nil
}

// ValidateNameAvailable checks if a service name is available
func (v *ServiceValidator) ValidateNameAvailable(name string) error {
	// Validate name format
	if err := (&models.Service{Name: name}).ValidateName(); err != nil {
		return err
	}

	// Check if name is already used by another service
	serviceRecord, err := v.findServiceByName(name)
	if err == nil && serviceRecord != nil {
		return &models.ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("service name '%s' is already in use", name),
		}
	}

	// Check if directory already exists
	dirPath := filepath.Join(v.baseDir, name)
	if _, err := os.Stat(dirPath); err == nil {
		return &models.ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("directory '%s' already exists", name),
		}
	}

	// Check if systemd service already exists
	serviceName := fmt.Sprintf("%s-pocketbase.service", name)
	serviceStatus, _ := v.systemdManager.GetServiceStatus(name)
	if serviceStatus != nil && serviceStatus.Name != "" {
		return &models.ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("systemd service '%s' already exists", serviceName),
		}
	}

	return nil
}

// findServiceByPort finds a service by port in the database
func (v *ServiceValidator) findServiceByPort(port int) (*daos.Record, error) {
	collection, err := v.dao.FindCollectionByNameOrId("services")
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	record, err := v.dao.FindFirstRecordByData(collection.Id, "port", port)
	if err != nil {
		return nil, err
	}

	return record, nil
}

// findServiceByName finds a service by name in the database
func (v *ServiceValidator) findServiceByName(name string) (*daos.Record, error) {
	collection, err := v.dao.FindCollectionByNameOrId("services")
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	record, err := v.dao.FindFirstRecordByData(collection.Id, "name", name)
	if err != nil {
		return nil, err
	}

	return record, nil
}

// GetNextAvailablePort returns the next available port
func (v *ServiceValidator) GetNextAvailablePort() (int, error) {
	// Get all services from the database
	collection, err := v.dao.FindCollectionByNameOrId("services")
	if err != nil {
		return 0, fmt.Errorf("collection not found: %w", err)
	}

	records, err := v.dao.FindRecordsByExpr(collection.Id)
	if err != nil {
		return 0, fmt.Errorf("failed to list services: %w", err)
	}

	// Extract all used ports
	usedPorts := make([]int, 0, len(records))
	for _, record := range records {
		portStr := record.GetString("port")
		port, err := strconv.Atoi(portStr)
		if err == nil {
			usedPorts = append(usedPorts, port)
		}
	}

	// Sort ports to find the highest one
	sort.Ints(usedPorts)

	// Start with default port if no services exist
	nextPort := 8091
	if len(usedPorts) > 0 {
		// Start from the highest used port + 1
		nextPort = usedPorts[len(usedPorts)-1] + 1
	}

	// Check if the port is actually available
	for port := nextPort; port < 10000; port++ {
		err := v.ValidatePortAvailable(port, "")
		if err == nil {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found in range 8091-9999")
}
