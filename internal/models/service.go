package models

import (
	"fmt"
	"regexp"
	"strings"
)

// Service represents a PocketBase service instance
type Service struct {
	ID        string `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Port      int    `json:"port" db:"port"`
	Version   string `json:"version" db:"version"`
	Subdomain string `json:"subdomain" db:"subdomain"`
	Status    string `json:"status" db:"status"`
	CreatedAt string `json:"created" db:"created"`
	UpdatedAt string `json:"updated" db:"updated"`
}

// ServiceValidation represents the validation status of a service
type ServiceValidation struct {
	SystemdExists   bool     `json:"systemd_exists"`
	SystemdRunning  bool     `json:"systemd_running"`
	CaddyConfigured bool     `json:"caddy_configured"`
	BinaryExists    bool     `json:"binary_exists"`
	PortMatches     bool     `json:"port_matches"`
	Issues          []string `json:"issues"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateName validates the service name according to requirements
func (s *Service) ValidateName() error {
	if s.Name == "" {
		return ValidationError{Field: "name", Message: "service name is required"}
	}

	if len(s.Name) > 50 {
		return ValidationError{Field: "name", Message: "service name must be 50 characters or less"}
	}

	// Check for valid characters (alphanumeric, underscore, hyphen)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(s.Name) {
		return ValidationError{Field: "name", Message: "service name can only contain letters, numbers, underscores, and hyphens"}
	}

	return nil
}

// ValidatePort validates the service port according to requirements
func (s *Service) ValidatePort() error {
	if s.Port < 8000 || s.Port > 9999 {
		return ValidationError{Field: "port", Message: "port must be between 8000 and 9999"}
	}

	return nil
}

// ValidateVersion validates the service version according to requirements
func (s *Service) ValidateVersion() error {
	if s.Version == "" {
		return ValidationError{Field: "version", Message: "version is required"}
	}

	// Check semantic version format (e.g., 1.2.3)
	validVersion := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !validVersion.MatchString(s.Version) {
		return ValidationError{Field: "version", Message: "version must be in semantic version format (e.g., 1.2.3)"}
	}

	return nil
}

// ValidateStatus validates the service status
func (s *Service) ValidateStatus() error {
	validStatuses := []string{"creating", "running", "stopped", "error"}

	for _, status := range validStatuses {
		if s.Status == status {
			return nil
		}
	}

	return ValidationError{Field: "status", Message: fmt.Sprintf("status must be one of: %s", strings.Join(validStatuses, ", "))}
}

// ValidateSubdomain validates the subdomain format
func (s *Service) ValidateSubdomain() error {
	if s.Subdomain == "" {
		return ValidationError{Field: "subdomain", Message: "subdomain is required"}
	}

	// Basic subdomain validation (alphanumeric and hyphens, no dots)
	validSubdomain := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !validSubdomain.MatchString(s.Subdomain) {
		return ValidationError{Field: "subdomain", Message: "subdomain can only contain letters, numbers, and hyphens"}
	}

	if len(s.Subdomain) > 63 {
		return ValidationError{Field: "subdomain", Message: "subdomain must be 63 characters or less"}
	}

	return nil
}

// Validate performs comprehensive validation of the service
func (s *Service) Validate() []error {
	var errors []error

	if err := s.ValidateName(); err != nil {
		errors = append(errors, err)
	}

	if err := s.ValidatePort(); err != nil {
		errors = append(errors, err)
	}

	if err := s.ValidateVersion(); err != nil {
		errors = append(errors, err)
	}

	if err := s.ValidateStatus(); err != nil {
		errors = append(errors, err)
	}

	if err := s.ValidateSubdomain(); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// IsValid returns true if the service passes all validation checks
func (s *Service) IsValid() bool {
	return len(s.Validate()) == 0
}
