package validation

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Validator handles validation of service configurations
type Validator struct {
	baseDir     string
	systemdDir  string
	caddyConfig string
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	IsValid  bool              `json:"is_valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// NewValidator creates a new validator
func NewValidator(baseDir, systemdDir, caddyConfig string) *Validator {
	return &Validator{
		baseDir:     baseDir,
		systemdDir:  systemdDir,
		caddyConfig: caddyConfig,
	}
}

// ValidateProjectName validates a project name
func (v *Validator) ValidateProjectName(name string) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check if name is empty
	if strings.TrimSpace(name) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "project_name",
			Message: "Project name cannot be empty",
			Code:    "EMPTY_NAME",
		})
		return result
	}

	// Check length
	if len(name) > 50 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "project_name",
			Message: "Project name cannot exceed 50 characters",
			Code:    "NAME_TOO_LONG",
		})
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "project_name",
			Message: "Project name can only contain letters, numbers, hyphens, and underscores",
			Code:    "INVALID_CHARACTERS",
		})
	}

	// Check if name starts with a letter
	if !regexp.MustCompile(`^[a-zA-Z]`).MatchString(name) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "project_name",
			Message: "Project name must start with a letter",
			Code:    "INVALID_START",
		})
	}

	// Check for reserved names
	reservedNames := []string{"admin", "api", "www", "mail", "ftp", "root", "system", "caddy", "systemd"}
	for _, reserved := range reservedNames {
		if strings.EqualFold(name, reserved) {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "project_name",
				Message: fmt.Sprintf("'%s' is a reserved name", reserved),
				Code:    "RESERVED_NAME",
			})
		}
	}

	return result
}

// ValidatePort validates a port number
func (v *Validator) ValidatePort(port int) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check port range
	if port < 1024 || port > 65535 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "port",
			Message: "Port must be between 1024 and 65535",
			Code:    "INVALID_PORT_RANGE",
		})
		return result
	}

	// Check if port is available
	if !v.IsPortAvailable(port) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("Port %d is already in use", port),
			Code:    "PORT_IN_USE",
		})
	}

	return result
}

// ValidateDomain validates a domain name
func (v *Validator) ValidateDomain(domain string) ValidationResult {
	result := ValidationResult{IsValid: true}

	if strings.TrimSpace(domain) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "domain",
			Message: "Domain cannot be empty",
			Code:    "EMPTY_DOMAIN",
		})
		return result
	}

	// Basic domain validation
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "domain",
			Message: "Invalid domain format",
			Code:    "INVALID_DOMAIN_FORMAT",
		})
	}

	return result
}

// ValidateVersion validates a PocketBase version
func (v *Validator) ValidateVersion(version string) ValidationResult {
	result := ValidationResult{IsValid: true}

	if strings.TrimSpace(version) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "version",
			Message: "Version cannot be empty",
			Code:    "EMPTY_VERSION",
		})
		return result
	}

	// Basic version format validation (semantic versioning)
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	if !versionRegex.MatchString(version) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "version",
			Message: "Invalid version format (expected semantic versioning like 1.2.3)",
			Code:    "INVALID_VERSION_FORMAT",
		})
	}

	return result
}

// IsPortAvailable checks if a port is available
func (v *Validator) IsPortAvailable(port int) bool {
	// Try to bind to the port
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// ValidateServiceUniqueness checks if a service name is unique
func (v *Validator) ValidateServiceUniqueness(projectName string, existingServices []string) ValidationResult {
	result := ValidationResult{IsValid: true}

	for _, existing := range existingServices {
		if strings.EqualFold(projectName, existing) {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "project_name",
				Message: fmt.Sprintf("Service '%s' already exists", projectName),
				Code:    "DUPLICATE_SERVICE",
			})
			break
		}
	}

	return result
}

// ValidatePortUniqueness checks if a port is unique among existing services
func (v *Validator) ValidatePortUniqueness(port int, existingPorts []int) ValidationResult {
	result := ValidationResult{IsValid: true}

	for _, existing := range existingPorts {
		if port == existing {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "port",
				Message: fmt.Sprintf("Port %d is already used by another service", port),
				Code:    "DUPLICATE_PORT",
			})
			break
		}
	}

	return result
}

// ValidateSystemRequirements checks system prerequisites
func (v *Validator) ValidateSystemRequirements() ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check if systemd is available
	if !v.isCommandAvailable("systemctl") {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "system",
			Message: "systemd is not available on this system",
			Code:    "MISSING_SYSTEMD",
		})
	}

	// Check if Caddy is available
	if !v.isCommandAvailable("caddy") {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "system",
			Message: "Caddy is not installed or not in PATH",
			Code:    "MISSING_CADDY",
		})
	}

	// Check if systemd directory exists and is writable
	if err := v.checkDirectoryWritable(v.systemdDir); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "system",
			Message: fmt.Sprintf("Cannot write to systemd directory: %v", err),
			Code:    "SYSTEMD_DIR_NOT_WRITABLE",
		})
	}

	// Check if Caddyfile exists and is writable
	if v.caddyConfig != "" {
		if err := v.checkFileWritable(v.caddyConfig); err != nil {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   "system",
				Message: fmt.Sprintf("Cannot write to Caddyfile: %v", err),
				Code:    "CADDYFILE_NOT_WRITABLE",
			})
		}
	}

	// Check base directory permissions
	if err := v.checkDirectoryWritable(v.baseDir); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "system",
			Message: fmt.Sprintf("Cannot write to base directory: %v", err),
			Code:    "BASE_DIR_NOT_WRITABLE",
		})
	}

	return result
}

// ValidateServiceConfiguration validates complete service configuration
func (v *Validator) ValidateServiceConfiguration(projectName string, port int, version string, domain string, existingServices []string, existingPorts []int) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Validate project name
	nameResult := v.ValidateProjectName(projectName)
	result.Errors = append(result.Errors, nameResult.Errors...)
	result.Warnings = append(result.Warnings, nameResult.Warnings...)
	if !nameResult.IsValid {
		result.IsValid = false
	}

	// Validate port
	portResult := v.ValidatePort(port)
	result.Errors = append(result.Errors, portResult.Errors...)
	result.Warnings = append(result.Warnings, portResult.Warnings...)
	if !portResult.IsValid {
		result.IsValid = false
	}

	// Validate version
	versionResult := v.ValidateVersion(version)
	result.Errors = append(result.Errors, versionResult.Errors...)
	result.Warnings = append(result.Warnings, versionResult.Warnings...)
	if !versionResult.IsValid {
		result.IsValid = false
	}

	// Validate domain
	domainResult := v.ValidateDomain(domain)
	result.Errors = append(result.Errors, domainResult.Errors...)
	result.Warnings = append(result.Warnings, domainResult.Warnings...)
	if !domainResult.IsValid {
		result.IsValid = false
	}

	// Validate uniqueness
	uniqueServiceResult := v.ValidateServiceUniqueness(projectName, existingServices)
	result.Errors = append(result.Errors, uniqueServiceResult.Errors...)
	if !uniqueServiceResult.IsValid {
		result.IsValid = false
	}

	uniquePortResult := v.ValidatePortUniqueness(port, existingPorts)
	result.Errors = append(result.Errors, uniquePortResult.Errors...)
	if !uniquePortResult.IsValid {
		result.IsValid = false
	}

	return result
}

// isCommandAvailable checks if a command is available in PATH
func (v *Validator) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// checkDirectoryWritable checks if a directory is writable
func (v *Validator) checkDirectoryWritable(dir string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Try to create a test file
	testFile := filepath.Join(dir, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// checkFileWritable checks if a file is writable
func (v *Validator) checkFileWritable(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}

	// Try to open for writing
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}
	file.Close()

	return nil
}

// ValidateExistingService validates the consistency of an existing service
func (v *Validator) ValidateExistingService(projectName string, expectedPort int, expectedDomain string) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check systemd service file
	systemdServicePath := filepath.Join(v.systemdDir, fmt.Sprintf("%s-pocketbase.service", projectName))
	if _, err := os.Stat(systemdServicePath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "systemd",
			Message: "SystemD service file does not exist",
			Code:    "MISSING_SYSTEMD_SERVICE",
		})
	} else {
		// Validate service file content
		content, err := os.ReadFile(systemdServicePath)
		if err == nil {
			portStr := strconv.Itoa(expectedPort)
			if !strings.Contains(string(content), portStr) {
				result.Warnings = append(result.Warnings, ValidationError{
					Field:   "systemd",
					Message: fmt.Sprintf("SystemD service file does not contain expected port %d", expectedPort),
					Code:    "SYSTEMD_PORT_MISMATCH",
				})
			}
		}
	}

	// Check service directory
	serviceDir := filepath.Join(v.baseDir, projectName)
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "service",
			Message: "Service directory does not exist",
			Code:    "MISSING_SERVICE_DIR",
		})
	}

	// Check PocketBase binary
	pbBinary := filepath.Join(serviceDir, "pocketbase")
	if _, err := os.Stat(pbBinary); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "service",
			Message: "PocketBase binary does not exist",
			Code:    "MISSING_POCKETBASE_BINARY",
		})
	}

	return result
}
