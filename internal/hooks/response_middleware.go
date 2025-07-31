package hooks

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tigawanna/pockestrator/internal/models"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ResponseMiddleware handles API response formatting and error handling
type ResponseMiddleware struct {
	logger services.LoggerService
}

// NewResponseMiddleware creates a new ResponseMiddleware
func NewResponseMiddleware(logger services.LoggerService) *ResponseMiddleware {
	return &ResponseMiddleware{
		logger: logger,
	}
}

// Register registers the middleware with the PocketBase app
func (m *ResponseMiddleware) Register(app core.App) error {
	// Add middleware to handle errors and format responses
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Add middleware to handle errors
		e.Router.Use(m.errorMiddleware)

		// Add middleware to handle config conflicts
		e.Router.Use(m.configConflictMiddleware)

		return nil
	})

	return nil
}

// errorMiddleware handles errors and formats responses
func (m *ResponseMiddleware) errorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err == nil {
			return nil
		}

		// Log the error
		m.logger.LogError(err)

		// Check if response was already sent
		if c.Response().Committed {
			return err
		}

		// Convert error to AppError
		var appErr *models.AppError
		if models.IsAppError(err) {
			appErr = err.(*models.AppError)
		} else {
			// Convert standard errors to AppError
			appErr = models.GetAppError(err)
		}

		// Create error response
		errorResponse := models.NewErrorResponse(appErr)

		// Determine HTTP status code based on error type
		statusCode := http.StatusInternalServerError
		switch appErr.Type {
		case models.ErrorTypeValidation:
			statusCode = http.StatusBadRequest
		case models.ErrorTypeConfiguration:
			statusCode = http.StatusBadRequest
		case models.ErrorTypeNetwork:
			statusCode = http.StatusServiceUnavailable
		case models.ErrorTypeSystem:
			statusCode = http.StatusInternalServerError
		case models.ErrorTypeRuntime:
			statusCode = http.StatusInternalServerError
		case models.ErrorTypeDatabase:
			statusCode = http.StatusInternalServerError
		}

		// Return JSON response
		return c.JSON(statusCode, errorResponse)
	}
}

// configConflictMiddleware handles configuration conflicts
func (m *ResponseMiddleware) configConflictMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Process the request
		err := next(c)
		if err != nil {
			return err
		}

		// Check if there's a config conflict in the context
		conflict, ok := c.Get("config_conflict").(*services.ConfigConflict)
		if !ok || conflict == nil || !conflict.HasConflict {
			return nil
		}

		// Get the original response
		originalBody := c.Response().Writer.(*apis.ResponseWriter).Body()

		// Parse the original response
		var originalResponse map[string]interface{}
		if err := json.Unmarshal(originalBody, &originalResponse); err != nil {
			m.logger.Error("Failed to parse original response: %v", err)
			return nil
		}

		// Add conflict information to the response
		originalResponse["config_conflict"] = conflict

		// Return the modified response
		return c.JSON(http.StatusOK, originalResponse)
	}
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*models.AppError); ok {
		// Create a new AppError with the same type but updated message
		return models.NewAppError(
			appErr.Type,
			appErr.Code,
			message+": "+appErr.Message,
			appErr.Severity,
		).WithDetails(appErr.Details).WithOriginalErr(appErr.OriginalErr)
	}

	// Create a new AppError with the original error
	return models.NewAppError(
		models.ErrorTypeUnknown,
		"wrapped_error",
		message+": "+err.Error(),
		models.SeverityError,
	).WithOriginalErr(err)
}
