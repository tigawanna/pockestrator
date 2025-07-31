package hooks

import (
	"fmt"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ConfigSyncHooks handles configuration synchronization hooks
type ConfigSyncHooks struct {
	configSync services.ConfigSyncService
	logger     services.LoggerService
}

// NewConfigSyncHooks creates a new ConfigSyncHooks instance
func NewConfigSyncHooks(configSync services.ConfigSyncService) *ConfigSyncHooks {
	return &ConfigSyncHooks{
		configSync: configSync,
		logger:     nil, // Will be set later
	}
}

// SetLogger sets the logger for the hooks
func (h *ConfigSyncHooks) SetLogger(logger services.LoggerService) {
	h.logger = logger
}

// RegisterConfigSyncEndpoints registers configuration sync endpoints
func (h *ConfigSyncHooks) RegisterConfigSyncEndpoints(app core.App) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Config sync status endpoint
		e.Router.AddRoute(apis.Route{
			Method:  "GET",
			Path:    "/api/services/:id/config-sync-status",
			Handler: h.getConfigSyncStatus,
			Middlewares: []apis.MiddlewareFunc{
				apis.RequireAdminAuth(),
			},
		})

		return nil
	})
}

// getConfigSyncStatus returns the configuration sync status for a service
func (h *ConfigSyncHooks) getConfigSyncStatus(c echo.Context) error {
	// Get service ID from path
	id := c.PathParam("id")
	if id == "" {
		return models.NewValidationError("missing_id", "Service ID is required")
	}

	// Get service record
	record, err := c.Get("app").(core.App).Dao().FindRecordById("services", id)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to find service record: %v", err)
		}
		return models.NewDatabaseError("record_not_found", fmt.Sprintf("Service with ID %s not found", id))
	}

	// Convert record to service model
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

	// Detect conflicts
	conflict, err := h.configSync.DetectConflicts(service)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to detect conflicts: %v", err)
		}
		return models.NewSystemError("conflict_detection_failed",
			fmt.Sprintf("Failed to detect configuration conflicts: %v", err),
			models.SeverityError)
	}

	// Return conflict status
	return c.JSON(200, conflict)
}
