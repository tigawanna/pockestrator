package models

import (
	"testing"
)

func TestService_ValidateName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "valid name with letters and numbers",
			serviceName: "myservice123",
			wantErr:     false,
		},
		{
			name:        "valid name with underscores",
			serviceName: "my_service",
			wantErr:     false,
		},
		{
			name:        "valid name with hyphens",
			serviceName: "my-service",
			wantErr:     false,
		},
		{
			name:        "valid name with mixed characters",
			serviceName: "my-service_123",
			wantErr:     false,
		},
		{
			name:        "empty name",
			serviceName: "",
			wantErr:     true,
			errMessage:  "service name is required",
		},
		{
			name:        "name too long",
			serviceName: "this-is-a-very-long-service-name-that-exceeds-fifty-characters-limit",
			wantErr:     true,
			errMessage:  "service name must be 50 characters or less",
		},
		{
			name:        "name with invalid characters",
			serviceName: "my.service",
			wantErr:     true,
			errMessage:  "service name can only contain letters, numbers, underscores, and hyphens",
		},
		{
			name:        "name with spaces",
			serviceName: "my service",
			wantErr:     true,
			errMessage:  "service name can only contain letters, numbers, underscores, and hyphens",
		},
		{
			name:        "name with special characters",
			serviceName: "my@service",
			wantErr:     true,
			errMessage:  "service name can only contain letters, numbers, underscores, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{Name: tt.serviceName}
			err := s.ValidateName()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateName() expected error but got none")
					return
				}
				if err.Error() != "name: "+tt.errMessage {
					t.Errorf("ValidateName() error = %v, want %v", err.Error(), "name: "+tt.errMessage)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateName() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestService_ValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "valid port 8000",
			port:    8000,
			wantErr: false,
		},
		{
			name:    "valid port 9999",
			port:    9999,
			wantErr: false,
		},
		{
			name:    "valid port 8080",
			port:    8080,
			wantErr: false,
		},
		{
			name:    "port too low",
			port:    7999,
			wantErr: true,
		},
		{
			name:    "port too high",
			port:    10000,
			wantErr: true,
		},
		{
			name:    "port zero",
			port:    0,
			wantErr: true,
		},
		{
			name:    "negative port",
			port:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{Port: tt.port}
			err := s.ValidatePort()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePort() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePort() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestService_ValidateVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "valid semantic version",
			version: "1.2.3",
			wantErr: false,
		},
		{
			name:    "valid version with larger numbers",
			version: "10.20.30",
			wantErr: false,
		},
		{
			name:    "valid version with zero",
			version: "0.1.0",
			wantErr: false,
		},
		{
			name:       "empty version",
			version:    "",
			wantErr:    true,
			errMessage: "version is required",
		},
		{
			name:       "invalid version format - missing patch",
			version:    "1.2",
			wantErr:    true,
			errMessage: "version must be in semantic version format (e.g., 1.2.3)",
		},
		{
			name:       "invalid version format - too many parts",
			version:    "1.2.3.4",
			wantErr:    true,
			errMessage: "version must be in semantic version format (e.g., 1.2.3)",
		},
		{
			name:       "invalid version format - non-numeric",
			version:    "1.2.a",
			wantErr:    true,
			errMessage: "version must be in semantic version format (e.g., 1.2.3)",
		},
		{
			name:       "invalid version format - with v prefix",
			version:    "v1.2.3",
			wantErr:    true,
			errMessage: "version must be in semantic version format (e.g., 1.2.3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{Version: tt.version}
			err := s.ValidateVersion()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateVersion() expected error but got none")
					return
				}
				if err.Error() != "version: "+tt.errMessage {
					t.Errorf("ValidateVersion() error = %v, want %v", err.Error(), "version: "+tt.errMessage)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateVersion() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestService_ValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{
			name:    "valid status creating",
			status:  "creating",
			wantErr: false,
		},
		{
			name:    "valid status running",
			status:  "running",
			wantErr: false,
		},
		{
			name:    "valid status stopped",
			status:  "stopped",
			wantErr: false,
		},
		{
			name:    "valid status error",
			status:  "error",
			wantErr: false,
		},
		{
			name:    "invalid status",
			status:  "invalid",
			wantErr: true,
		},
		{
			name:    "empty status",
			status:  "",
			wantErr: true,
		},
		{
			name:    "case sensitive status",
			status:  "Running",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{Status: tt.status}
			err := s.ValidateStatus()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateStatus() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateStatus() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestService_ValidateSubdomain(t *testing.T) {
	tests := []struct {
		name       string
		subdomain  string
		wantErr    bool
		errMessage string
	}{
		{
			name:      "valid subdomain",
			subdomain: "myservice",
			wantErr:   false,
		},
		{
			name:      "valid subdomain with numbers",
			subdomain: "service123",
			wantErr:   false,
		},
		{
			name:      "valid subdomain with hyphens",
			subdomain: "my-service",
			wantErr:   false,
		},
		{
			name:       "empty subdomain",
			subdomain:  "",
			wantErr:    true,
			errMessage: "subdomain is required",
		},
		{
			name:       "subdomain with dots",
			subdomain:  "my.service",
			wantErr:    true,
			errMessage: "subdomain can only contain letters, numbers, and hyphens",
		},
		{
			name:       "subdomain with underscores",
			subdomain:  "my_service",
			wantErr:    true,
			errMessage: "subdomain can only contain letters, numbers, and hyphens",
		},
		{
			name:       "subdomain too long",
			subdomain:  "this-is-a-very-long-subdomain-name-that-exceeds-sixty-three-chars",
			wantErr:    true,
			errMessage: "subdomain must be 63 characters or less",
		},
		{
			name:       "subdomain with special characters",
			subdomain:  "my@service",
			wantErr:    true,
			errMessage: "subdomain can only contain letters, numbers, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{Subdomain: tt.subdomain}
			err := s.ValidateSubdomain()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateSubdomain() expected error but got none")
					return
				}
				if err.Error() != "subdomain: "+tt.errMessage {
					t.Errorf("ValidateSubdomain() error = %v, want %v", err.Error(), "subdomain: "+tt.errMessage)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSubdomain() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestService_Validate(t *testing.T) {
	tests := []struct {
		name         string
		service      Service
		wantErrCount int
	}{
		{
			name: "valid service",
			service: Service{
				Name:      "myservice",
				Port:      8080,
				Version:   "1.2.3",
				Status:    "running",
				Subdomain: "myservice",
			},
			wantErrCount: 0,
		},
		{
			name: "service with multiple validation errors",
			service: Service{
				Name:      "",
				Port:      7000,
				Version:   "",
				Status:    "invalid",
				Subdomain: "",
			},
			wantErrCount: 5,
		},
		{
			name: "service with some validation errors",
			service: Service{
				Name:      "valid-name",
				Port:      7000, // invalid
				Version:   "1.2.3",
				Status:    "invalid", // invalid
				Subdomain: "valid-subdomain",
			},
			wantErrCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.service.Validate()
			if len(errors) != tt.wantErrCount {
				t.Errorf("Validate() returned %d errors, want %d", len(errors), tt.wantErrCount)
			}
		})
	}
}

func TestService_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		want    bool
	}{
		{
			name: "valid service",
			service: Service{
				Name:      "myservice",
				Port:      8080,
				Version:   "1.2.3",
				Status:    "running",
				Subdomain: "myservice",
			},
			want: true,
		},
		{
			name: "invalid service",
			service: Service{
				Name:      "",
				Port:      7000,
				Version:   "",
				Status:    "invalid",
				Subdomain: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.service.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "name",
		Message: "service name is required",
	}

	expected := "name: service name is required"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}
