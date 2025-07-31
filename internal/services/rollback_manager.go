package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tigawanna/pockestrator/internal/models"
)

// RollbackOperation represents a rollback operation
type RollbackOperation struct {
	Type        string
	Description string
	Execute     func() error
}

// RollbackManager handles rollback operations for failed service operations
type RollbackManager struct {
	operations []RollbackOperation
	baseDir    string
}

// NewRollbackManager creates a new RollbackManager
func NewRollbackManager(baseDir string) *RollbackManager {
	return &RollbackManager{
		operations: make([]RollbackOperation, 0),
		baseDir:    baseDir,
	}
}

// AddOperation adds a rollback operation to the stack
func (rm *RollbackManager) AddOperation(opType string, description string, execute func() error) {
	rm.operations = append(rm.operations, RollbackOperation{
		Type:        opType,
		Description: description,
		Execute:     execute,
	})
}

// Rollback executes all rollback operations in reverse order
func (rm *RollbackManager) Rollback() error {
	if len(rm.operations) == 0 {
		return nil
	}

	log.Printf("Starting rollback of %d operations", len(rm.operations))

	var lastErr error
	// Execute operations in reverse order (LIFO)
	for i := len(rm.operations) - 1; i >= 0; i-- {
		op := rm.operations[i]
		log.Printf("Rolling back operation: %s - %s", op.Type, op.Description)

		if err := op.Execute(); err != nil {
			log.Printf("Error during rollback of %s: %v", op.Description, err)
			lastErr = err
			// Continue with other rollback operations even if one fails
		}
	}

	// Clear operations after rollback
	rm.operations = make([]RollbackOperation, 0)

	if lastErr != nil {
		return fmt.Errorf("errors occurred during rollback: %w", lastErr)
	}

	return nil
}

// Clear clears all rollback operations without executing them
func (rm *RollbackManager) Clear() {
	rm.operations = make([]RollbackOperation, 0)
}

// AddSystemdServiceRollback adds a rollback operation for systemd service creation
func (rm *RollbackManager) AddSystemdServiceRollback(systemdManager SystemdManager, serviceName string) {
	rm.AddOperation("systemd", fmt.Sprintf("Remove systemd service %s", serviceName), func() error {
		return systemdManager.RemoveService(serviceName)
	})
}

// AddCaddyConfigRollback adds a rollback operation for Caddy configuration
func (rm *RollbackManager) AddCaddyConfigRollback(caddyManager CaddyManager, serviceName string) {
	rm.AddOperation("caddy", fmt.Sprintf("Remove Caddy configuration for %s", serviceName), func() error {
		if err := caddyManager.RemoveConfiguration(serviceName); err != nil {
			return err
		}
		return caddyManager.ReloadCaddy()
	})
}

// AddDirectoryRollback adds a rollback operation for directory creation
func (rm *RollbackManager) AddDirectoryRollback(serviceName string) {
	rm.AddOperation("directory", fmt.Sprintf("Remove directory for %s", serviceName), func() error {
		projectDir := filepath.Join(rm.baseDir, serviceName)
		return os.RemoveAll(projectDir)
	})
}

// AddFileRollback adds a rollback operation for file creation
func (rm *RollbackManager) AddFileRollback(filePath string) {
	rm.AddOperation("file", fmt.Sprintf("Remove file %s", filePath), func() error {
		return os.Remove(filePath)
	})
}

// ServiceCreationManager handles the creation of services with rollback support
type ServiceCreationManager struct {
	pbManager       PocketBaseManager
	systemdManager  SystemdManager
	caddyManager    CaddyManager
	rollbackMgr     *RollbackManager
	baseDir         string
	defaultEmail    string
	defaultPassword string
}

// NewServiceCreationManager creates a new ServiceCreationManager
func NewServiceCreationManager(
	pbManager PocketBaseManager,
	systemdManager SystemdManager,
	caddyManager CaddyManager,
	baseDir string,
	defaultEmail string,
	defaultPassword string,
) *ServiceCreationManager {
	return &ServiceCreationManager{
		pbManager:       pbManager,
		systemdManager:  systemdManager,
		caddyManager:    caddyManager,
		rollbackMgr:     NewRollbackManager(baseDir),
		baseDir:         baseDir,
		defaultEmail:    defaultEmail,
		defaultPassword: defaultPassword,
	}
}

// CreateService creates a new service with rollback support
func (scm *ServiceCreationManager) CreateService(service *models.Service) error {
	// Clear any previous rollback operations
	scm.rollbackMgr.Clear()

	log.Printf("Setting up new service: %s (port: %d)", service.Name, service.Port)

	// 1. Create project directory
	projectDir := filepath.Join(scm.baseDir, service.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	scm.rollbackMgr.AddDirectoryRollback(service.Name)

	// 2. Download PocketBase
	log.Printf("Downloading PocketBase version %s for %s", service.Version, service.Name)
	if err := scm.pbManager.DownloadPocketBase(service.Version, service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to download PocketBase: %w", err)
	}
	zipPath := filepath.Join(projectDir, fmt.Sprintf("pocketbase_%s.zip", service.Version))
	scm.rollbackMgr.AddFileRollback(zipPath)

	// 3. Extract PocketBase
	log.Printf("Extracting PocketBase for %s", service.Name)
	if err := scm.pbManager.ExtractPocketBase(service.Name, service.Version); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to extract PocketBase: %w", err)
	}
	binaryPath := filepath.Join(projectDir, "pocketbase")
	scm.rollbackMgr.AddFileRollback(binaryPath)

	// 4. Set permissions
	log.Printf("Setting permissions for %s", service.Name)
	if err := scm.pbManager.SetPermissions(service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// 5. Create required directories
	dirs := []string{
		filepath.Join(projectDir, "pb_data"),
		filepath.Join(projectDir, "pb_public"),
		filepath.Join(projectDir, "pb_migrations"),
		filepath.Join(projectDir, "pb_hooks"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			scm.rollbackMgr.Rollback()
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 6. Create systemd service
	log.Printf("Creating systemd service for %s", service.Name)
	if err := scm.systemdManager.CreateService(service); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to create systemd service: %w", err)
	}
	scm.rollbackMgr.AddSystemdServiceRollback(scm.systemdManager, service.Name)

	// 7. Enable systemd service
	log.Printf("Enabling systemd service for %s", service.Name)
	if err := scm.systemdManager.EnableService(service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to enable systemd service: %w", err)
	}

	// 8. Add Caddy configuration
	log.Printf("Adding Caddy configuration for %s", service.Name)
	if err := scm.caddyManager.AddConfiguration(service); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to add Caddy configuration: %w", err)
	}
	scm.rollbackMgr.AddCaddyConfigRollback(scm.caddyManager, service.Name)

	// 9. Start the service
	log.Printf("Starting service %s", service.Name)
	if err := scm.systemdManager.StartService(service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to start service: %w", err)
	}

	// 10. Create default superuser (with a delay to allow service to start)
	log.Printf("Creating superuser for %s", service.Name)
	if err := scm.pbManager.CreateSuperUser(service.Name, scm.defaultEmail, scm.defaultPassword); err != nil {
		log.Printf("Warning: Failed to create superuser: %v", err)
		// Don't fail the entire process if superuser creation fails
	}

	// Clear rollback operations on success
	scm.rollbackMgr.Clear()

	log.Printf("Service %s setup completed successfully", service.Name)
	return nil
}

// UpdateService updates an existing service with rollback support
func (scm *ServiceCreationManager) UpdateService(service *models.Service, oldService *models.Service) error {
	// Clear any previous rollback operations
	scm.rollbackMgr.Clear()

	log.Printf("Updating service: %s", service.Name)

	// Check what has changed and update accordingly
	configChanged := service.Port != oldService.Port || service.Subdomain != oldService.Subdomain

	if configChanged {
		// 1. Update systemd service file if port changed
		if service.Port != oldService.Port {
			log.Printf("Updating systemd service for %s (port changed: %d -> %d)",
				service.Name, oldService.Port, service.Port)

			// Save old service configuration for rollback
			scm.rollbackMgr.AddOperation("systemd", fmt.Sprintf("Restore systemd service for %s", service.Name), func() error {
				return scm.systemdManager.CreateService(oldService)
			})

			if err := scm.systemdManager.CreateService(service); err != nil {
				scm.rollbackMgr.Rollback()
				return fmt.Errorf("failed to update systemd service: %w", err)
			}
		}

		// 2. Update Caddy configuration if port or subdomain changed
		if service.Port != oldService.Port || service.Subdomain != oldService.Subdomain {
			log.Printf("Updating Caddy configuration for %s", service.Name)

			// Save old Caddy configuration for rollback
			scm.rollbackMgr.AddOperation("caddy", fmt.Sprintf("Restore Caddy configuration for %s", service.Name), func() error {
				if err := scm.caddyManager.AddConfiguration(oldService); err != nil {
					return err
				}
				return scm.caddyManager.ReloadCaddy()
			})

			if err := scm.caddyManager.AddConfiguration(service); err != nil {
				scm.rollbackMgr.Rollback()
				return fmt.Errorf("failed to update Caddy configuration: %w", err)
			}
		}

		// 3. Restart the service if it was running and configuration changed
		if oldService.Status == "running" {
			log.Printf("Restarting service %s after configuration change", service.Name)

			// Save service status for rollback
			wasRunning := oldService.Status == "running"
			scm.rollbackMgr.AddOperation("service", fmt.Sprintf("Restore service status for %s", service.Name), func() error {
				if wasRunning {
					return scm.systemdManager.StartService(service.Name)
				}
				return scm.systemdManager.StopService(service.Name)
			})

			if err := scm.systemdManager.StopService(service.Name); err != nil {
				log.Printf("Warning: Failed to stop service: %v", err)
			}

			if err := scm.systemdManager.StartService(service.Name); err != nil {
				scm.rollbackMgr.Rollback()
				return fmt.Errorf("failed to restart service: %w", err)
			}
		}
	} else if service.Status != oldService.Status {
		// Handle status changes
		if service.Status == "running" && oldService.Status != "running" {
			log.Printf("Starting service %s", service.Name)

			// Save service status for rollback
			scm.rollbackMgr.AddOperation("service", fmt.Sprintf("Restore service status for %s", service.Name), func() error {
				return scm.systemdManager.StopService(service.Name)
			})

			if err := scm.systemdManager.StartService(service.Name); err != nil {
				scm.rollbackMgr.Rollback()
				return fmt.Errorf("failed to start service: %w", err)
			}
		} else if service.Status == "stopped" && oldService.Status != "stopped" {
			log.Printf("Stopping service %s", service.Name)

			// Save service status for rollback
			scm.rollbackMgr.AddOperation("service", fmt.Sprintf("Restore service status for %s", service.Name), func() error {
				return scm.systemdManager.StartService(service.Name)
			})

			if err := scm.systemdManager.StopService(service.Name); err != nil {
				scm.rollbackMgr.Rollback()
				return fmt.Errorf("failed to stop service: %w", err)
			}
		}
	}

	// Clear rollback operations on success
	scm.rollbackMgr.Clear()

	log.Printf("Service %s updated successfully", service.Name)
	return nil
}

// DeleteService deletes a service with rollback support
func (scm *ServiceCreationManager) DeleteService(service *models.Service) error {
	// Clear any previous rollback operations
	scm.rollbackMgr.Clear()

	log.Printf("Deleting service: %s", service.Name)

	// 1. Stop and remove systemd service
	log.Printf("Removing systemd service for %s", service.Name)

	// Save systemd service for rollback
	scm.rollbackMgr.AddOperation("systemd", fmt.Sprintf("Restore systemd service for %s", service.Name), func() error {
		if err := scm.systemdManager.CreateService(service); err != nil {
			return err
		}
		if err := scm.systemdManager.EnableService(service.Name); err != nil {
			return err
		}
		if service.Status == "running" {
			return scm.systemdManager.StartService(service.Name)
		}
		return nil
	})

	if err := scm.systemdManager.RemoveService(service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to remove systemd service: %w", err)
	}

	// 2. Remove Caddy configuration
	log.Printf("Removing Caddy configuration for %s", service.Name)

	// Save Caddy configuration for rollback
	scm.rollbackMgr.AddOperation("caddy", fmt.Sprintf("Restore Caddy configuration for %s", service.Name), func() error {
		if err := scm.caddyManager.AddConfiguration(service); err != nil {
			return err
		}
		return scm.caddyManager.ReloadCaddy()
	})

	if err := scm.caddyManager.RemoveConfiguration(service.Name); err != nil {
		scm.rollbackMgr.Rollback()
		return fmt.Errorf("failed to remove Caddy configuration: %w", err)
	}

	// 3. Remove service directory (optional, might want to keep for backup)
	// We'll add this as a rollback operation but not execute it by default
	projectDir := filepath.Join(scm.baseDir, service.Name)
	log.Printf("Service directory will be preserved at: %s", projectDir)

	// Clear rollback operations on success
	scm.rollbackMgr.Clear()

	log.Printf("Service %s deleted successfully", service.Name)
	return nil
}
