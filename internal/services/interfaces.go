package services

import (
	"github.com/tigawanna/pockestrator/internal/models"
)

// PocketBaseManager handles PocketBase binary management
type PocketBaseManager interface {
	DownloadPocketBase(version string, projectName string) error
	ExtractPocketBase(projectName string, version string) error
	SetPermissions(projectName string) error
	CreateSuperUser(projectName string, email string, password string) error
}

// SystemdManager handles systemd service management
type SystemdManager interface {
	CreateService(service *models.Service) error
	EnableService(serviceName string) error
	StartService(serviceName string) error
	StopService(serviceName string) error
	RemoveService(serviceName string) error
	IsServiceRunning(serviceName string) bool
	GetServiceStatus(serviceName string) (*ServiceStatus, error)
}

// CaddyManager handles Caddy configuration management
type CaddyManager interface {
	AddConfiguration(service *models.Service) error
	RemoveConfiguration(serviceName string) error
	ReloadCaddy() error
	ValidateConfiguration(service *models.Service) bool
}

// ValidationService handles service validation
type ValidationService interface {
	ValidateService(service *models.Service) (*models.ServiceValidation, error)
	ValidatePortAvailable(port int, excludeService string) error
	ValidateNameAvailable(name string) error
	GetNextAvailablePort() (int, error)
}

// ConfigSyncService handles bidirectional synchronization between collections and system files
type ConfigSyncService interface {
	// SyncServiceToSystem synchronizes a service record to system files (collection → files)
	SyncServiceToSystem(service *models.Service) error

	// SyncSystemToService synchronizes system files to a service record (files → collection)
	SyncSystemToService(service *models.Service) (*models.Service, error)

	// DetectConflicts checks for conflicts between a service record and system files
	DetectConflicts(service *models.Service) (*ConfigConflict, error)

	// ValidateServiceConfig validates that a service configuration is consistent with system files
	ValidateServiceConfig(service *models.Service) (bool, []string)
}

// ConfigConflict represents a conflict between collection data and system files
type ConfigConflict struct {
	ServiceID      string            `json:"service_id"`
	ServiceName    string            `json:"service_name"`
	HasConflict    bool              `json:"has_conflict"`
	ConflictFields map[string]string `json:"conflict_fields"`
	SystemState    *models.Service   `json:"system_state,omitempty"`
}

// ServiceStatus represents the status of a systemd service
type ServiceStatus struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// LoggerService provides logging functionality
type LoggerService interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warning(format string, v ...interface{})
	Error(format string, v ...interface{})
	Fatal(format string, v ...interface{})
	LogError(err error)
	SetLogLevel(level LogLevel)
	GetLogLevel() LogLevel
}
