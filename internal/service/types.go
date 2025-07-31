package service

import (
	"time"
)

// ServiceConfig represents a PocketBase service configuration
type ServiceConfig struct {
	ID                string    `json:"id"`
	ProjectName       string    `json:"project_name"`
	Port              int       `json:"port"`
	PocketBaseVersion string    `json:"pocketbase_version"`
	Domain            string    `json:"domain"`
	Status            string    `json:"status"` // active, inactive, error
	SystemdConfigHash string    `json:"systemd_config_hash"`
	CaddyConfigHash   string    `json:"caddy_config_hash"`
	LastHealthCheck   time.Time `json:"last_health_check"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DeploymentConfig holds configuration for service deployment
type DeploymentConfig struct {
	ServiceConfig
	BaseDir         string `json:"base_dir"`
	SystemdDir      string `json:"systemd_dir"`
	CaddyConfigPath string `json:"caddy_config_path"`
	SuperuserEmail  string `json:"superuser_email"`
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	ServiceID     string    `json:"service_id"`
	IsRunning     bool      `json:"is_running"`
	SystemdStatus string    `json:"systemd_status"`
	CaddyStatus   string    `json:"caddy_status"`
	ConfigMatch   bool      `json:"config_match"`
	LastChecked   time.Time `json:"last_checked"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	SystemdValid bool     `json:"systemd_valid"`
	CaddyValid   bool     `json:"caddy_valid"`
	PortValid    bool     `json:"port_valid"`
}
