package hooks

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ServiceHooks handles service lifecycle hooks
type ServiceHooks struct {
	configSync      services.ConfigSyncService
	pbManager       services.PocketBaseManager
	systemdManager  services.SystemdManager
	caddyManager    services.CaddyManager
	validator       services.ValidationService
	baseDir         string
	defaultEmail    string
	defaultPassword string
	logger          services.LoggerService
}

// NewServiceHooks creates a new ServiceHooks instance
func NewServiceHooks(
	configSync services.ConfigSyncService,
	pbManager services.PocketBaseManager,
	systemdManager services.SystemdManager,
	caddyManager services.CaddyManager,
	validator services.ValidationService,
	baseDir string,
	defaultEmail string,
	defaultPassword string,
) *ServiceHooks {
	return &ServiceHooks{
		configSync:      configSync,
		pbManager:       pbManager,
		systemdManager:  systemdManager,
		caddyManager:    caddyManager,
		validator:       validator,
		baseDir:         baseDir,
		defaultEmail:    defaultEmail,
		defaultPassword: defaultPassword,
		logger:          nil, // Will be set later via SetLogger
	}
}

// RegisterHooks registers all service hooks with PocketBase
func (h *ServiceHooks) RegisterHooks(app core.App) error {
	// Register validation middleware for services collection
	app.OnRecordBeforeRequestEvent("services", h.validateConfigBeforeRequest)

	// Register lifecycle hooks for services collection
	app.OnRecordBeforeCreateEvent("services", h.validateBeforeCreate)
	app.OnRecordAfterCreateEvent("services", h.onServiceCreate)
	app.OnRecordBeforeUpdateEvent("services", h.validateBeforeUpdate)
	app.OnRecordAfterUpdateEvent("services", h.onServiceUpdate)
	app.OnRecordBeforeDeleteEvent("services", h.validateBeforeDelete)
	app.OnRecordAfterDeleteEvent("services", h.onServiceDelete)

	return nil
}

// validateConfigBeforeRequest validates service configuration before returning to client
func (h *ServiceHooks) validateConfigBeforeRequest(e *core.RecordRequestEvent) error {
	// Skip validation for create requests (service doesn't exist yet)
	if e.HttpContext.Request().Method == "POST" {
		return nil
	}

	// Skip validation for delete requests
	if e.HttpContext.Request().Method == "DELETE" {
		return nil
	}

	// Get the service record
	record := e.Record
	if record == nil {
		return nil
	}

	// Convert PocketBase record to our Service model
	service, err := recordToService(record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert record to service: %v", err)
		} else {
			log.Printf("Failed to convert record to service: %v", err)
		}
		return nil // Don't block the request, just log the error
	}

	// Detect conflicts between record and system files
	conflict, err := h.configSync.DetectConflicts(service)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to detect conflicts for service %s: %v", service.Name, err)
		} else {
			log.Printf("Failed to detect conflicts: %v", err)
		}
		return nil // Don't block the request, just log the error
	}

	// If there are conflicts, add them to the response context
	if conflict.HasConflict {
		// Store conflict information in the context for the response handler
		e.HttpContext.Set("config_conflict", conflict)

		// Check if auto-sync is enabled via query parameter
		autoSync := e.HttpContext.QueryParam("auto_sync")
		if autoSync == "system_to_db" {
			// Sync system files to database
			updatedService, err := h.configSync.SyncSystemToService(service)
			if err != nil {
				if h.logger != nil {
					h.logger.Error("Failed to sync system to service %s: %v", service.Name, err)
				} else {
					log.Printf("Failed to sync system to service: %v", err)
				}
				return nil // Don't block the request, just log the error
			}

			// Update the record with the synced values
			updateRecordFromService(record, updatedService)

			if h.logger != nil {
				h.logger.Info("Successfully synced system configuration to database for service %s", service.Name)
			}
		} else if autoSync == "db_to_system" {
			// Sync database to system files
			err := h.configSync.SyncServiceToSystem(service)
			if err != nil {
				if h.logger != nil {
					h.logger.Error("Failed to sync service %s to system: %v", service.Name, err)
				} else {
					log.Printf("Failed to sync service to system: %v", err)
				}
				return nil // Don't block the request, just log the error
			}

			if h.logger != nil {
				h.logger.Info("Successfully synced database configuration to system for service %s", service.Name)
			}
		}
	}

	return nil
}

// validateBeforeCreate validates a service before creation
func (h *ServiceHooks) validateBeforeCreate(e *core.RecordCreateEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		return fmt.Errorf("invalid service data: %w", err)
	}

	// Validate service model
	if errs := service.Validate(); len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs[0])
	}

	// Validate name uniqueness
	if err := h.validator.ValidateNameAvailable(service.Name); err != nil {
		return fmt.Errorf("name validation failed: %w", err)
	}

	// If port is not specified or is 0, assign the next available port
	if service.Port == 0 {
		nextPort, err := h.validator.GetNextAvailablePort()
		if err != nil {
			return fmt.Errorf("failed to get next available port: %w", err)
		}
		service.Port = nextPort
		e.Record.Set("port", nextPort)
	} else {
		// Validate port availability
		if err := h.validator.ValidatePortAvailable(service.Port, ""); err != nil {
			return fmt.Errorf("port validation failed: %w", err)
		}
	}

	// Set initial status to "creating"
	e.Record.Set("status", "creating")

	return nil
}

// onServiceCreate handles the complete service setup after creation
func (h *ServiceHooks) onServiceCreate(e *core.RecordEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert record to service: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert record to service: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Creating new service: %s (port: %d, version: %s)",
			service.Name, service.Port, service.Version)
	}

	// Create a service creation manager with rollback support
	creationManager := services.NewServiceCreationManager(
		h.pbManager,
		h.systemdManager,
		h.caddyManager,
		h.baseDir,
		h.defaultEmail,
		h.defaultPassword,
	)

	// Create the service with rollback support
	if err := creationManager.CreateService(service); err != nil {
		e.Record.Set("status", "error")
		if h.logger != nil {
			h.logger.Error("Failed to create service %s: %v", service.Name, err)
		}
		return models.NewSystemError("service_creation_failed",
			fmt.Sprintf("Failed to create service %s", service.Name),
			models.SeverityCritical).WithOriginalErr(err)
	}

	// Update service status to "running"
	e.Record.Set("status", "running")

	if h.logger != nil {
		h.logger.Info("Service %s created successfully", service.Name)
	}

	return nil
}

// validateBeforeUpdate validates a service before update
func (h *ServiceHooks) validateBeforeUpdate(e *core.RecordUpdateEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Invalid service data during update: %v", err)
		}
		return models.NewValidationError("invalid_service_data",
			fmt.Sprintf("Invalid service data: %v", err))
	}

	// Validate service model
	if errs := service.Validate(); len(errs) > 0 {
		if h.logger != nil {
			h.logger.Error("Service validation failed during update: %v", errs[0])
		}
		return models.NewValidationError("validation_failed",
			fmt.Sprintf("Validation failed: %v", errs[0]))
	}

	// Check if port has changed
	oldService, err := recordToService(e.OriginalRecord)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert original record during update: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert original record: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	// If port has changed, validate the new port
	if service.Port != oldService.Port {
		if h.logger != nil {
			h.logger.Info("Port change detected for service %s: %d -> %d",
				service.Name, oldService.Port, service.Port)
		}

		if err := h.validator.ValidatePortAvailable(service.Port, service.ID); err != nil {
			if h.logger != nil {
				h.logger.Error("Port validation failed during update: %v", err)
			}
			return models.NewValidationError("port_validation_failed",
				fmt.Sprintf("Port validation failed: %v", err))
		}
	}

	return nil
}

// onServiceUpdate handles configuration synchronization after update
func (h *ServiceHooks) onServiceUpdate(e *core.RecordEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert record to service during update: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert record to service: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	// Get original service data
	oldService, err := recordToService(e.OriginalRecord)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert original record during update: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert original record: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Updating service: %s (port: %d -> %d, subdomain: %s -> %s)",
			service.Name,
			oldService.Port, service.Port,
			oldService.Subdomain, service.Subdomain)
	}

	// Create a service creation manager with rollback support
	creationManager := services.NewServiceCreationManager(
		h.pbManager,
		h.systemdManager,
		h.caddyManager,
		h.baseDir,
		h.defaultEmail,
		h.defaultPassword,
	)

	// Update the service with rollback support
	if err := creationManager.UpdateService(service, oldService); err != nil {
		e.Record.Set("status", "error")
		if h.logger != nil {
			h.logger.Error("Failed to update service %s: %v", service.Name, err)
		}
		return models.NewSystemError("service_update_failed",
			fmt.Sprintf("Failed to update service %s", service.Name),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Service %s updated successfully", service.Name)
	}

	return nil
}

// validateBeforeDelete validates a service before deletion
func (h *ServiceHooks) validateBeforeDelete(e *core.RecordDeleteEvent) error {
	// No specific validation needed before deletion
	// You could add checks here if needed (e.g., dependencies)
	return nil
}

// onServiceDelete handles cleanup operations after service deletion
func (h *ServiceHooks) onServiceDelete(e *core.RecordEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert record to service during deletion: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert record to service: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Deleting service: %s (port: %d)", service.Name, service.Port)
	}

	// Create a service creation manager with rollback support
	creationManager := services.NewServiceCreationManager(
		h.pbManager,
		h.systemdManager,
		h.caddyManager,
		h.baseDir,
		h.defaultEmail,
		h.defaultPassword,
	)

	// Delete the service with rollback support
	if err := creationManager.DeleteService(service); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to delete service %s: %v", service.Name, err)
		}
		return models.NewSystemError("service_deletion_failed",
			fmt.Sprintf("Failed to delete service %s", service.Name),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Service %s deleted successfully", service.Name)
	}

	return nil
}

// syncServiceToSystem syncs service record to system files after create/update
func (h *ServiceHooks) syncServiceToSystem(e *core.RecordEvent) error {
	// Convert PocketBase record to our Service model
	service, err := recordToService(e.Record)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to convert record to service during sync: %v", err)
		}
		return models.NewSystemError("record_conversion_failed",
			fmt.Sprintf("Failed to convert record to service: %v", err),
			models.SeverityError).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Syncing service %s configuration to system files", service.Name)
	}

	// Sync service to system files
	if err := h.configSync.SyncServiceToSystem(service); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to sync service %s to system: %v", service.Name, err)
		}
		return models.NewConfigurationError("sync_failed",
			fmt.Sprintf("Failed to sync service %s to system files", service.Name)).WithOriginalErr(err)
	}

	if h.logger != nil {
		h.logger.Info("Successfully synced service %s configuration to system files", service.Name)
	}

	return nil
}

// Helper functions

// recordToService converts a PocketBase record to a Service model
func recordToService(record *pbmodels.Record) (*models.Service, error) {
	if record == nil {
		return nil, fmt.Errorf("record is nil")
	}

	service := &models.Service{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Port:      int(record.GetInt("port")),
		Version:   record.GetString("version"),
		Subdomain: record.GetString("subdomain"),
		Status:    record.GetString("status"),
		CreatedAt: record.GetString("created"),
		UpdatedAt: record.GetString("updated"),
	}

	return service, nil
}

// updateRecordFromService updates a PocketBase record from a Service model
func updateRecordFromService(record *pbmodels.Record, service *models.Service) {
	if record == nil || service == nil {
		return
	}

	record.Set("name", service.Name)
	record.Set("port", service.Port)
	record.Set("version", service.Version)
	record.Set("subdomain", service.Subdomain)
	record.Set("status", service.Status)
}
