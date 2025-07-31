package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ServiceHandler handles service-related HTTP requests
type ServiceHandler struct {
	validator services.ValidationService
}

// NewServiceHandler creates a new ServiceHandler
func NewServiceHandler(validator services.ValidationService) *ServiceHandler {
	return &ServiceHandler{
		validator: validator,
	}
}

// ValidateService handles GET /api/services/{id}/validate
func (h *ServiceHandler) ValidateService(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Convert PocketBase record to our Service model
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

	// Validate service configuration
	validation, err := h.validator.ValidateService(service)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to validate service", err)
	}

	// Return validation result
	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"service":    service,
		"validation": validation,
	})
}

// GetServiceLogs handles GET /api/services/{id}/logs
func (h *ServiceHandler) GetServiceLogs(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Get log file path
	logPath := fmt.Sprintf("/home/ubuntu/%s/service.log", serviceName)

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
			"logs":    "",
			"message": "No logs found for this service",
		})
	}

	// Read log file (last 100 lines)
	cmd := exec.Command("tail", "-n", "100", logPath)
	output, err := cmd.Output()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to read log file", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"logs": string(output),
	})
}

// RestartService handles POST /api/services/{id}/restart
func (h *ServiceHandler) RestartService(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Create systemd manager
	systemdManager := services.NewSystemdManager()

	// Stop the service
	if err := systemdManager.StopService(serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to stop service", err)
	}

	// Start the service
	if err := systemdManager.StartService(serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to start service", err)
	}

	// Get service status
	status, err := systemdManager.GetServiceStatus(serviceName)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get service status", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Service restarted successfully",
		"status":  status,
	})
}

// GetAvailablePorts handles GET /api/system/ports/available
func (h *ServiceHandler) GetAvailablePorts(e *core.RequestEvent) error {
	// Get count parameter (number of ports to return)
	countStr := e.HttpContext.QueryParam("count")
	count := 5 // Default to 5 ports
	if countStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	// Get starting port parameter
	startPortStr := e.HttpContext.QueryParam("start")
	startPort := 8091 // Default starting port
	if startPortStr != "" {
		if parsedPort, err := strconv.Atoi(startPortStr); err == nil && parsedPort >= 8000 {
			startPort = parsedPort
		}
	}

	// Get next available port from validator
	nextPort, err := h.validator.GetNextAvailablePort()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get next available port", err)
	}

	// If the next available port is higher than the requested start port, use it
	if nextPort > startPort {
		startPort = nextPort
	}

	// Find available ports
	availablePorts := make([]int, 0, count)
	currentPort := startPort

	for len(availablePorts) < count && currentPort < 10000 {
		// Check if port is available
		err := h.validator.ValidatePortAvailable(currentPort, "")
		if err == nil {
			availablePorts = append(availablePorts, currentPort)
		}
		currentPort++
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"next_available_port": nextPort,
		"available_ports":     availablePorts,
		"count":               len(availablePorts),
	})
}

// UploadServiceFile handles POST /api/services/{id}/upload
func (h *ServiceHandler) UploadServiceFile(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Get directory parameter (pb_public, pb_migrations, pb_hooks)
	directory := e.HttpContext.QueryParam("directory")
	if directory == "" {
		directory = "pb_public" // Default to pb_public
	}

	// Validate directory
	validDirs := map[string]bool{
		"pb_public":     true,
		"pb_migrations": true,
		"pb_hooks":      true,
	}
	if !validDirs[directory] {
		return apis.NewBadRequestError("Invalid directory. Must be one of: pb_public, pb_migrations, pb_hooks", nil)
	}

	// Get file from form
	file, err := e.HttpContext.FormFile("file")
	if err != nil {
		return apis.NewBadRequestError("Missing or invalid file", err)
	}

	// Create service directory if it doesn't exist
	baseDir := fmt.Sprintf("/home/ubuntu/%s", serviceName)
	dirPath := filepath.Join(baseDir, directory)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to create directory", err)
	}

	// Save file
	filePath := filepath.Join(dirPath, file.Filename)
	src, err := file.Open()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to open uploaded file", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to create destination file", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to save file", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "File uploaded successfully",
		"file": map[string]interface{}{
			"name":      file.Filename,
			"size":      file.Size,
			"directory": directory,
			"path":      filePath,
		},
	})
}

// ListServiceFiles handles GET /api/services/{id}/files
func (h *ServiceHandler) ListServiceFiles(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Get directory parameter (pb_public, pb_migrations, pb_hooks)
	directory := e.HttpContext.QueryParam("directory")
	if directory == "" {
		directory = "pb_public" // Default to pb_public
	}

	// Validate directory
	validDirs := map[string]bool{
		"pb_public":     true,
		"pb_migrations": true,
		"pb_hooks":      true,
	}
	if !validDirs[directory] {
		return apis.NewBadRequestError("Invalid directory. Must be one of: pb_public, pb_migrations, pb_hooks", nil)
	}

	// Get service directory
	baseDir := fmt.Sprintf("/home/ubuntu/%s", serviceName)
	dirPath := filepath.Join(baseDir, directory)

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory doesn't exist, return empty list
		return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
			"files": []interface{}{},
		})
	}

	// Read directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to read directory", err)
	}

	// Build file list
	fileList := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}

		info, err := file.Info()
		if err != nil {
			continue // Skip files with errors
		}

		fileList = append(fileList, map[string]interface{}{
			"name":      file.Name(),
			"size":      info.Size(),
			"modified":  info.ModTime().Format(time.RFC3339),
			"directory": directory,
			"path":      filepath.Join(dirPath, file.Name()),
		})
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"files": fileList,
	})
}

// DeleteServiceFile handles DELETE /api/services/{id}/files/{filename}
func (h *ServiceHandler) DeleteServiceFile(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get filename from path
	filename := e.Params["filename"]
	if filename == "" {
		return apis.NewBadRequestError("Filename is required", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Get directory parameter (pb_public, pb_migrations, pb_hooks)
	directory := e.HttpContext.QueryParam("directory")
	if directory == "" {
		directory = "pb_public" // Default to pb_public
	}

	// Validate directory
	validDirs := map[string]bool{
		"pb_public":     true,
		"pb_migrations": true,
		"pb_hooks":      true,
	}
	if !validDirs[directory] {
		return apis.NewBadRequestError("Invalid directory. Must be one of: pb_public, pb_migrations, pb_hooks", nil)
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return apis.NewBadRequestError("Invalid filename", nil)
	}

	// Get file path
	baseDir := fmt.Sprintf("/home/ubuntu/%s", serviceName)
	filePath := filepath.Join(baseDir, directory, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return apis.NewNotFoundError("File not found", nil)
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to delete file", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "File deleted successfully",
	})
}

// SyncServiceConfig handles POST /api/services/{id}/sync-config
func (h *ServiceHandler) SyncServiceConfig(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Convert PocketBase record to our Service model
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

	// Get sync direction from query parameter
	direction := e.HttpContext.QueryParam("direction")
	if direction == "" {
		direction = "db_to_system" // Default direction
	}

	// Create systemd and caddy managers
	systemdManager := services.NewSystemdManager()
	caddyManager := services.NewCaddyManager("example.com") // Replace with actual domain

	// Create a repository adapter for the PocketBase DAO
	repo := &PocketBaseServiceRepository{dao: e.App.Dao()}

	// Create config sync service
	configSync := services.NewConfigSyncService(
		systemdManager,
		caddyManager,
		"/home/ubuntu",
		"/lib/systemd/system",
		"/etc/caddy/Caddyfile",
		"example.com", // Replace with actual domain
		repo,
	)

	var result interface{}
	var syncErr error

	// Perform sync based on direction
	if direction == "system_to_db" {
		// Sync system files to database
		updatedService, err := configSync.SyncSystemToService(service)
		if err != nil {
			syncErr = fmt.Errorf("failed to sync system to service: %w", err)
		} else {
			// Update record with synced values
			record.Set("name", updatedService.Name)
			record.Set("port", updatedService.Port)
			record.Set("version", updatedService.Version)
			record.Set("subdomain", updatedService.Subdomain)
			record.Set("status", updatedService.Status)

			// Save updated record
			if err := e.App.Dao().SaveRecord(record); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Failed to save updated record", err)
			}

			result = updatedService
		}
	} else if direction == "db_to_system" {
		// Sync database to system files
		if err := configSync.SyncServiceToSystem(service); err != nil {
			syncErr = fmt.Errorf("failed to sync service to system: %w", err)
		} else {
			result = map[string]interface{}{
				"success": true,
				"message": "Service configuration synced to system",
			}
		}
	} else {
		return apis.NewBadRequestError("Invalid sync direction. Must be 'system_to_db' or 'db_to_system'", nil)
	}

	// Handle sync error
	if syncErr != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Sync failed", syncErr)
	}

	// Return result
	return e.HttpContext.JSON(http.StatusOK, result)
}

// ListPocketBaseVersions handles GET /api/pocketbase/versions
func (h *ServiceHandler) ListPocketBaseVersions(e *core.RequestEvent) error {
	// Create PocketBase manager
	pbManager := services.NewPocketBaseManager()

	// Get available versions
	versions, err := pbManager.GetAvailableVersions()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get available versions", err)
	}

	// Get latest version
	latestVersion, err := pbManager.GetLatestVersion()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get latest version", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"versions": versions,
		"latest":   latestVersion,
	})
}

// UpdatePocketBase handles POST /api/services/{id}/update-pocketbase
func (h *ServiceHandler) UpdatePocketBase(e *core.RequestEvent) error {
	// Get service ID from path
	id := e.Params["id"]
	if id == "" {
		return apis.NewNotFoundError("Service not found", nil)
	}

	// Get service from database
	record, err := e.App.Dao().FindRecordById("services", id)
	if err != nil {
		return apis.NewNotFoundError("Service not found", err)
	}

	// Get service name
	serviceName := record.GetString("name")
	if serviceName == "" {
		return apis.NewBadRequestError("Invalid service record", nil)
	}

	// Get version parameter
	version := e.HttpContext.FormValue("version")
	if version == "" {
		// If no version is specified, use latest
		pbManager := services.NewPocketBaseManager()
		latestVersion, err := pbManager.GetLatestVersion()
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to get latest version", err)
		}
		version = latestVersion
	}

	// Create systemd manager
	systemdManager := services.NewSystemdManager()

	// Stop the service
	if err := systemdManager.StopService(serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to stop service", err)
	}

	// Create PocketBase manager
	pbManager := services.NewPocketBaseManager()

	// Download and extract PocketBase
	if err := pbManager.DownloadPocketBase(version, serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to download PocketBase", err)
	}

	if err := pbManager.ExtractPocketBase(serviceName, version); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to extract PocketBase", err)
	}

	if err := pbManager.SetPermissions(serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to set permissions", err)
	}

	// Update version in database
	record.Set("version", version)
	if err := e.App.Dao().SaveRecord(record); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to update version in database", err)
	}

	// Start the service
	if err := systemdManager.StartService(serviceName); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to start service", err)
	}

	// Get service status
	status, err := systemdManager.GetServiceStatus(serviceName)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get service status", err)
	}

	return e.HttpContext.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("PocketBase updated to version %s", version),
		"version": version,
		"status":  status,
	})
}

// PocketBaseServiceRepository is an adapter that implements the services.ServiceRepository interface
// using PocketBase's DAO
type PocketBaseServiceRepository struct {
	dao *daos.Dao
}

// FindServiceByID finds a service by ID
func (r *PocketBaseServiceRepository) FindServiceByID(id string) (*models.Service, error) {
	record, err := r.dao.FindRecordById("services", id)
	if err != nil {
		return nil, err
	}
	return &models.Service{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Port:      int(record.GetInt("port")),
		Version:   record.GetString("version"),
		Subdomain: record.GetString("subdomain"),
		Status:    record.GetString("status"),
		CreatedAt: record.GetString("created"),
		UpdatedAt: record.GetString("updated"),
	}, nil
}

// FindServiceByName finds a service by name
func (r *PocketBaseServiceRepository) FindServiceByName(name string) (*models.Service, error) {
	record, err := r.dao.FindFirstRecordByData("services", "name", name)
	if err != nil {
		return nil, err
	}
	return &models.Service{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Port:      int(record.GetInt("port")),
		Version:   record.GetString("version"),
		Subdomain: record.GetString("subdomain"),
		Status:    record.GetString("status"),
		CreatedAt: record.GetString("created"),
		UpdatedAt: record.GetString("updated"),
	}, nil
}

// ListAllServices lists all services
func (r *PocketBaseServiceRepository) ListAllServices() ([]*models.Service, error) {
	records, err := r.dao.FindRecordsByExpr("services")
	if err != nil {
		return nil, err
	}

	services := make([]*models.Service, 0, len(records))
	for _, record := range records {
		services = append(services, &models.Service{
			ID:        record.Id,
			Name:      record.GetString("name"),
			Port:      int(record.GetInt("port")),
			Version:   record.GetString("version"),
			Subdomain: record.GetString("subdomain"),
			Status:    record.GetString("status"),
			CreatedAt: record.GetString("created"),
			UpdatedAt: record.GetString("updated"),
		})
	}

	return services, nil
}

// UpdateService updates a service
func (r *PocketBaseServiceRepository) UpdateService(service *models.Service) error {
	record, err := r.dao.FindRecordById("services", service.ID)
	if err != nil {
		return err
	}

	record.Set("name", service.Name)
	record.Set("port", service.Port)
	record.Set("version", service.Version)
	record.Set("subdomain", service.Subdomain)
	record.Set("status", service.Status)

	return r.dao.SaveRecord(record)
}
