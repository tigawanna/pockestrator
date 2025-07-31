package hooks

import (
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// SetLogger sets the logger for the hooks
func (h *ServiceHooks) SetLogger(logger services.LoggerService) {
	h.logger = logger
}

// logError logs an error with appropriate context
func (h *ServiceHooks) logError(err error, format string, v ...interface{}) {
	if h.logger == nil {
		return
	}

	// Create a service-specific error
	if appErr, ok := err.(*models.AppError); ok {
		h.logger.LogError(appErr)
	} else {
		// Create a new app error and log it
		appErr := models.NewSystemError(
			"service_hook_error",
			format,
			models.SeverityError,
		).WithOriginalErr(err)
		h.logger.LogError(appErr)
	}
}
