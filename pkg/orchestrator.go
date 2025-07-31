package pkg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tigawanna/pockestrator/internal/caddy"
	"github.com/tigawanna/pockestrator/internal/database"
	"github.com/tigawanna/pockestrator/internal/service"
	"github.com/tigawanna/pockestrator/internal/systemd"
	"github.com/tigawanna/pockestrator/internal/validation"
)

// Orchestrator coordinates all service management operations
type Orchestrator struct {
	serviceManager *service.Manager
	systemdManager *systemd.Manager
	caddyManager   *caddy.Manager
	validator      *validation.Validator
	dbManager      *database.Manager
	config         *Config
}

// Config holds orchestrator configuration
type Config struct {
	BaseDir       string
	SystemdDir    string
	CaddyConfig   string
	DefaultDomain string
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(
	serviceManager *service.Manager,
	systemdManager *systemd.Manager,
	caddyManager *caddy.Manager,
	validator *validation.Validator,
	dbManager *database.Manager,
	config *Config,
) *Orchestrator {
	return &Orchestrator{
		serviceManager: serviceManager,
		systemdManager: systemdManager,
		caddyManager:   caddyManager,
		validator:      validator,
		dbManager:      dbManager,
		config:         config,
	}
}

// ServiceRequest represents a service creation request
type ServiceRequest struct {
	ProjectName       string `json:"project_name"`
	PocketBaseVersion string `json:"pocketbase_version,omitempty"`
	Port              int    `json:"port,omitempty"`
	Domain            string `json:"domain,omitempty"`
	Description       string `json:"description,omitempty"`
	CreatedBy         string `json:"created_by,omitempty"`
}

// ServiceResponse represents a service operation response
type ServiceResponse struct {
	ID      string                       `json:"id"`
	Status  string                       `json:"status"`
	Message string                       `json:"message"`
	Data    *database.ServiceRecord      `json:"data,omitempty"`
	Errors  []validation.ValidationError `json:"errors,omitempty"`
}

// ServiceLogsResponse represents a service logs response
type ServiceLogsResponse struct {
	ServiceID   string    `json:"service_id"`
	ProjectName string    `json:"project_name"`
	Lines       int       `json:"lines"`
	Logs        []string  `json:"logs"`
	Timestamp   time.Time `json:"timestamp"`
}

// CreateService creates and deploys a new PocketBase service
func (o *Orchestrator) CreateService(ctx context.Context, req *ServiceRequest) (*ServiceResponse, error) {
	// Set defaults
	if req.PocketBaseVersion == "" {
		version, err := o.serviceManager.GetLatestVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest version: %w", err)
		}
		req.PocketBaseVersion = version
	}

	if req.Domain == "" {
		req.Domain = o.config.DefaultDomain
	}

	// Get existing services and ports for validation
	existingServices, err := o.dbManager.GetExistingServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing services: %w", err)
	}

	usedPorts, err := o.dbManager.GetUsedPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get used ports: %w", err)
	}

	// Auto-assign port if not provided
	if req.Port == 0 {
		req.Port = o.serviceManager.GetNextPort(usedPorts)
	}

	// Validate the service configuration
	validationResult := o.validator.ValidateServiceConfiguration(
		req.ProjectName,
		req.Port,
		req.PocketBaseVersion,
		req.Domain,
		existingServices,
		usedPorts,
	)

	if !validationResult.IsValid {
		return &ServiceResponse{
			Status:  "error",
			Message: "Validation failed",
			Errors:  validationResult.Errors,
		}, nil
	}

	// Create service record
	serviceRecord := &database.ServiceRecord{
		ProjectName:       req.ProjectName,
		Port:              req.Port,
		PocketBaseVersion: req.PocketBaseVersion,
		Domain:            req.Domain,
		Status:            "deploying",
		CreatedBy:         req.CreatedBy,
		LastHealthCheck:   time.Now(),
	}

	if err := o.dbManager.CreateService(ctx, serviceRecord); err != nil {
		return nil, fmt.Errorf("failed to create service record: %w", err)
	}

	// Deploy the service asynchronously
	go func() {
		if err := o.deployServiceAsync(context.Background(), serviceRecord); err != nil {
			// Update status to error
			o.dbManager.UpdateServiceStatus(context.Background(), serviceRecord.ID, "error")
		}
	}()

	return &ServiceResponse{
		ID:      serviceRecord.ID,
		Status:  "deploying",
		Message: "Service deployment started",
		Data:    serviceRecord,
	}, nil
}

// deployServiceAsync deploys a service asynchronously
func (o *Orchestrator) deployServiceAsync(ctx context.Context, serviceRecord *database.ServiceRecord) error {
	// Create deployment config
	deployConfig := &service.DeploymentConfig{
		ServiceConfig: service.ServiceConfig{
			ProjectName:       serviceRecord.ProjectName,
			Port:              serviceRecord.Port,
			PocketBaseVersion: serviceRecord.PocketBaseVersion,
			Domain:            serviceRecord.Domain,
		},
		BaseDir:         o.config.BaseDir,
		SystemdDir:      o.config.SystemdDir,
		CaddyConfigPath: o.config.CaddyConfig,
		SuperuserEmail:  fmt.Sprintf("admin@%s.%s", serviceRecord.ProjectName, serviceRecord.Domain),
	}

	// Deploy PocketBase instance
	if err := o.serviceManager.Deploy(ctx, deployConfig); err != nil {
		return fmt.Errorf("failed to deploy PocketBase: %w", err)
	}

	// Create systemd service
	systemdConfig := &systemd.ServiceConfig{
		ProjectName: serviceRecord.ProjectName,
		ServiceDir:  fmt.Sprintf("%s/%s", o.config.BaseDir, serviceRecord.ProjectName),
		Port:        serviceRecord.Port,
	}

	if err := o.systemdManager.CreateService(systemdConfig); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Enable and start systemd service
	if err := o.systemdManager.EnableService(serviceRecord.ProjectName); err != nil {
		return fmt.Errorf("failed to enable systemd service: %w", err)
	}

	// Wait for service to start
	time.Sleep(5 * time.Second)

	// Add Caddy configuration
	caddyConfig := &caddy.ServiceConfig{
		Subdomain: serviceRecord.ProjectName,
		Domain:    serviceRecord.Domain,
		Port:      serviceRecord.Port,
	}

	if err := o.caddyManager.AddService(caddyConfig); err != nil {
		return fmt.Errorf("failed to add Caddy configuration: %w", err)
	}

	// Reload Caddy
	if err := o.caddyManager.ReloadConfig(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	// Update service to active
	if err := o.dbManager.UpdateServiceStatus(ctx, serviceRecord.ID, "active"); err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	// Generate and store configuration hashes
	systemdContent, _ := o.generateSystemdConfig(systemdConfig)
	caddyContent, _ := o.generateCaddyConfig(caddyConfig)

	systemdHash := database.GenerateConfigHash(systemdContent)
	caddyHash := database.GenerateConfigHash(caddyContent)

	if err := o.dbManager.UpdateConfigHashes(ctx, serviceRecord.ID, systemdHash, caddyHash); err != nil {
		return fmt.Errorf("failed to update config hashes: %w", err)
	}

	return nil
}

// GetService retrieves a service by ID
func (o *Orchestrator) GetService(ctx context.Context, id string) (*ServiceResponse, error) {
	service, err := o.dbManager.GetService(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &ServiceResponse{
		ID:      service.ID,
		Status:  "success",
		Message: "Service retrieved successfully",
		Data:    service,
	}, nil
}

// ListServices retrieves all services
func (o *Orchestrator) ListServices(ctx context.Context) ([]*database.ServiceRecord, error) {
	return o.dbManager.ListServices(ctx)
}

// DeleteService removes a service completely
func (o *Orchestrator) DeleteService(ctx context.Context, id string) (*ServiceResponse, error) {
	// Get service record
	serviceRecord, err := o.dbManager.GetService(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Remove systemd service
	if err := o.systemdManager.RemoveService(serviceRecord.ProjectName); err != nil {
		return nil, fmt.Errorf("failed to remove systemd service: %w", err)
	}

	// Remove Caddy configuration
	if err := o.caddyManager.RemoveService(serviceRecord.ProjectName, serviceRecord.Domain); err != nil {
		return nil, fmt.Errorf("failed to remove Caddy configuration: %w", err)
	}

	// Reload Caddy
	if err := o.caddyManager.ReloadConfig(); err != nil {
		return nil, fmt.Errorf("failed to reload Caddy: %w", err)
	}

	// Remove service files
	serviceConfig := &service.ServiceConfig{
		ProjectName: serviceRecord.ProjectName,
		Port:        serviceRecord.Port,
		Domain:      serviceRecord.Domain,
	}

	if err := o.serviceManager.Remove(ctx, serviceConfig); err != nil {
		return nil, fmt.Errorf("failed to remove service files: %w", err)
	}

	// Delete database record
	if err := o.dbManager.DeleteService(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete service record: %w", err)
	}

	return &ServiceResponse{
		ID:      id,
		Status:  "success",
		Message: "Service deleted successfully",
	}, nil
}

// ControlService controls service operations (start/stop/restart)
func (o *Orchestrator) ControlService(ctx context.Context, id, action string) (*ServiceResponse, error) {
	service, err := o.dbManager.GetService(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	switch action {
	case "start":
		err = o.serviceManager.Start(service.ProjectName)
	case "stop":
		err = o.serviceManager.Stop(service.ProjectName)
	case "restart":
		err = o.serviceManager.Restart(service.ProjectName)
	default:
		return &ServiceResponse{
			Status:  "error",
			Message: fmt.Sprintf("Invalid action: %s", action),
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to %s service: %w", action, err)
	}

	// Update service status
	newStatus := "active"
	if action == "stop" {
		newStatus = "inactive"
	}

	if err := o.dbManager.UpdateServiceStatus(ctx, id, newStatus); err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	return &ServiceResponse{
		ID:      id,
		Status:  "success",
		Message: fmt.Sprintf("Service %s completed successfully", action),
	}, nil
}

// GetServiceStatus retrieves current service status
func (o *Orchestrator) GetServiceStatus(ctx context.Context, id string) (*service.HealthStatus, error) {
	serviceRecord, err := o.dbManager.GetService(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return o.serviceManager.GetServiceStatus(serviceRecord.ProjectName)
}

// ValidateSystemRequirements validates system prerequisites
func (o *Orchestrator) ValidateSystemRequirements() *validation.ValidationResult {
	result := o.validator.ValidateSystemRequirements()
	return &result
}

// GetUsedPorts retrieves all used ports from existing services
func (o *Orchestrator) GetUsedPorts(ctx context.Context) ([]int, error) {
	return o.dbManager.GetUsedPorts(ctx)
}

// ValidateServiceConfiguration validates a service configuration
func (o *Orchestrator) ValidateServiceConfiguration(
	projectName string,
	port int,
	pocketbaseVersion string,
	domain string,
	existingServices []*database.ServiceRecord,
	usedPorts []int,
) *validation.ValidationResult {
	// Convert service records to project names
	existingNames := make([]string, len(existingServices))
	for i, service := range existingServices {
		existingNames[i] = service.ProjectName
	}

	result := o.validator.ValidateServiceConfiguration(
		projectName,
		port,
		pocketbaseVersion,
		domain,
		existingNames,
		usedPorts,
	)

	return &result
}

// generateSystemdConfig generates systemd configuration content
func (o *Orchestrator) generateSystemdConfig(config *systemd.ServiceConfig) (string, error) {
	// This would generate the actual systemd config content
	return fmt.Sprintf("systemd config for %s on port %d", config.ProjectName, config.Port), nil
}

// generateCaddyConfig generates Caddy configuration content
func (o *Orchestrator) generateCaddyConfig(config *caddy.ServiceConfig) (string, error) {
	// This would generate the actual Caddy config content
	return fmt.Sprintf("caddy config for %s.%s on port %d", config.Subdomain, config.Domain, config.Port), nil
}

// GetServiceLogs retrieves service logs
func (o *Orchestrator) GetServiceLogs(ctx context.Context, serviceID string, lines int) (*ServiceLogsResponse, error) {
	// Get service record from database
	serviceRecord, err := o.dbManager.GetService(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Get logs from systemd
	logsText, err := o.systemdManager.GetServiceLogs(serviceRecord.ProjectName, lines)
	if err != nil {
		return nil, fmt.Errorf("failed to get service logs: %w", err)
	}

	// Split logs into lines
	logLines := strings.Split(strings.TrimSpace(logsText), "\n")

	return &ServiceLogsResponse{
		ServiceID:   serviceID,
		ProjectName: serviceRecord.ProjectName,
		Lines:       lines,
		Logs:        logLines,
		Timestamp:   time.Now(),
	}, nil
}
