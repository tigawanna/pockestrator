package services

import "github.com/tigawanna/pockestrator/internal/models"

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

// ServiceStatus represents the status of a systemd service
type ServiceStatus struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}
