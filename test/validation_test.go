package validation_test

import (
	"testing"

	"github.com/tigawanna/pockestrator/internal/validation"
)

func TestValidateProjectName(t *testing.T) {
	validator := validation.NewValidator("/tmp", "/tmp", "/tmp/Caddyfile")

	tests := []struct {
		name        string
		projectName string
		expectValid bool
		expectError string
	}{
		{
			name:        "Valid project name",
			projectName: "myproject",
			expectValid: true,
		},
		{
			name:        "Valid project name with numbers",
			projectName: "project123",
			expectValid: true,
		},
		{
			name:        "Valid project name with hyphen",
			projectName: "my-project",
			expectValid: true,
		},
		{
			name:        "Valid project name with underscore",
			projectName: "my_project",
			expectValid: true,
		},
		{
			name:        "Empty project name",
			projectName: "",
			expectValid: false,
			expectError: "EMPTY_NAME",
		},
		{
			name:        "Project name too long",
			projectName: "thisprojectnameiswaytoolongandexceedsthefiftycharacterlimit",
			expectValid: false,
			expectError: "NAME_TOO_LONG",
		},
		{
			name:        "Project name with invalid characters",
			projectName: "project@name",
			expectValid: false,
			expectError: "INVALID_CHARACTERS",
		},
		{
			name:        "Project name starting with number",
			projectName: "123project",
			expectValid: false,
			expectError: "INVALID_START",
		},
		{
			name:        "Reserved name",
			projectName: "admin",
			expectValid: false,
			expectError: "RESERVED_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateProjectName(tt.projectName)

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s, but not found in %+v", tt.expectError, result.Errors)
				}
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	validator := validation.NewValidator("/tmp", "/tmp", "/tmp/Caddyfile")

	tests := []struct {
		name        string
		port        int
		expectValid bool
		expectError string
	}{
		{
			name:        "Valid port",
			port:        8080,
			expectValid: true,
		},
		{
			name:        "Port too low",
			port:        80,
			expectValid: false,
			expectError: "INVALID_PORT_RANGE",
		},
		{
			name:        "Port too high",
			port:        70000,
			expectValid: false,
			expectError: "INVALID_PORT_RANGE",
		},
		{
			name:        "Valid high port",
			port:        65535,
			expectValid: true,
		},
		{
			name:        "Valid low port",
			port:        1024,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePort(tt.port)

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s, but not found in %+v", tt.expectError, result.Errors)
				}
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	validator := validation.NewValidator("/tmp", "/tmp", "/tmp/Caddyfile")

	tests := []struct {
		name        string
		version     string
		expectValid bool
		expectError string
	}{
		{
			name:        "Valid semantic version",
			version:     "1.2.3",
			expectValid: true,
		},
		{
			name:        "Valid version with prerelease",
			version:     "1.2.3-alpha.1",
			expectValid: true,
		},
		{
			name:        "Valid version with build metadata",
			version:     "1.2.3+20231201",
			expectValid: true,
		},
		{
			name:        "Empty version",
			version:     "",
			expectValid: false,
			expectError: "EMPTY_VERSION",
		},
		{
			name:        "Invalid version format",
			version:     "1.2",
			expectValid: false,
			expectError: "INVALID_VERSION_FORMAT",
		},
		{
			name:        "Invalid version format with text",
			version:     "latest",
			expectValid: false,
			expectError: "INVALID_VERSION_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateVersion(tt.version)

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s, but not found in %+v", tt.expectError, result.Errors)
				}
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	validator := validation.NewValidator("/tmp", "/tmp", "/tmp/Caddyfile")

	tests := []struct {
		name        string
		domain      string
		expectValid bool
		expectError string
	}{
		{
			name:        "Valid domain",
			domain:      "example.com",
			expectValid: true,
		},
		{
			name:        "Valid subdomain",
			domain:      "api.example.com",
			expectValid: true,
		},
		{
			name:        "Valid complex domain",
			domain:      "my-app.tigawanna.vip",
			expectValid: true,
		},
		{
			name:        "Empty domain",
			domain:      "",
			expectValid: false,
			expectError: "EMPTY_DOMAIN",
		},
		{
			name:        "Invalid domain format",
			domain:      "invalid..domain",
			expectValid: false,
			expectError: "INVALID_DOMAIN_FORMAT",
		},
		{
			name:        "Domain with invalid characters",
			domain:      "domain_with_underscore.com",
			expectValid: false,
			expectError: "INVALID_DOMAIN_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateDomain(tt.domain)

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s, but not found in %+v", tt.expectError, result.Errors)
				}
			}
		})
	}
}

func TestValidateServiceUniqueness(t *testing.T) {
	validator := validation.NewValidator("/tmp", "/tmp", "/tmp/Caddyfile")
	existingServices := []string{"service1", "service2", "Service3"}

	tests := []struct {
		name        string
		projectName string
		expectValid bool
		expectError string
	}{
		{
			name:        "Unique service name",
			projectName: "newservice",
			expectValid: true,
		},
		{
			name:        "Duplicate service name",
			projectName: "service1",
			expectValid: false,
			expectError: "DUPLICATE_SERVICE",
		},
		{
			name:        "Case insensitive duplicate",
			projectName: "SERVICE1",
			expectValid: false,
			expectError: "DUPLICATE_SERVICE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateServiceUniqueness(tt.projectName, existingServices)

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s, but not found in %+v", tt.expectError, result.Errors)
				}
			}
		})
	}
}
