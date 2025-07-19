package validators

import "github.com/tigawanna/pockestrator/internal/models"

// ServiceValidator provides validation functionality for services
type ServiceValidator struct {
	// TODO: Add dependencies for validation
}

// NewServiceValidator creates a new ServiceValidator
func NewServiceValidator() *ServiceValidator {
	return &ServiceValidator{}
}

// ValidateService validates a service configuration
func (v *ServiceValidator) ValidateService(service *models.Service) (*models.ServiceValidation, error) {
	// TODO: Implement service validation logic
	return &models.ServiceValidation{
		SystemdExists:   false,
		SystemdRunning:  false,
		CaddyConfigured: false,
		BinaryExists:    false,
		PortMatches:     false,
		Issues:          []string{"validation not implemented"},
	}, nil
}

// ValidatePortAvailable checks if a port is available
func (v *ServiceValidator) ValidatePortAvailable(port int, excludeService string) error {
	// TODO: Implement port availability validation
	return nil
}

// ValidateNameAvailable checks if a service name is available
func (v *ServiceValidator) ValidateNameAvailable(name string) error {
	// TODO: Implement name availability validation
	return nil
}

// GetNextAvailablePort returns the next available port
func (v *ServiceValidator) GetNextAvailablePort() (int, error) {
	// TODO: Implement next available port logic
	return 8091, nil
}
