package models

import (
	"encoding/json"
	"fmt"
)

// ErrorType represents the category of an error
type ErrorType string

const (
	// Error types
	ErrorTypeValidation    ErrorType = "validation"    // Validation errors (invalid input, etc.)
	ErrorTypeSystem        ErrorType = "system"        // System errors (file system, permissions, etc.)
	ErrorTypeNetwork       ErrorType = "network"       // Network errors (download failures, etc.)
	ErrorTypeConfiguration ErrorType = "configuration" // Configuration errors (invalid config, etc.)
	ErrorTypeRuntime       ErrorType = "runtime"       // Runtime errors (service failures, etc.)
	ErrorTypeDatabase      ErrorType = "database"      // Database errors
	ErrorTypeUnknown       ErrorType = "unknown"       // Unknown errors
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	// Error severities
	SeverityFatal    ErrorSeverity = "fatal"    // Fatal errors that require immediate attention
	SeverityCritical ErrorSeverity = "critical" // Critical errors that may impact functionality
	SeverityError    ErrorSeverity = "error"    // Standard errors
	SeverityWarning  ErrorSeverity = "warning"  // Warnings that don't prevent operation
	SeverityInfo     ErrorSeverity = "info"     // Informational messages
)

// AppError represents a structured application error
type AppError struct {
	Type        ErrorType     `json:"type"`
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Severity    ErrorSeverity `json:"severity"`
	Details     interface{}   `json:"details,omitempty"`
	OriginalErr error         `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// WithOriginalErr adds the original error
func (e *AppError) WithOriginalErr(err error) *AppError {
	e.OriginalErr = err
	return e
}

// MarshalJSON implements json.Marshaler
func (e *AppError) MarshalJSON() ([]byte, error) {
	type Alias AppError
	return json.Marshal(&struct {
		*Alias
		OriginalErr string `json:"original_error,omitempty"`
	}{
		Alias: (*Alias)(e),
		OriginalErr: func() string {
			if e.OriginalErr != nil {
				return e.OriginalErr.Error()
			}
			return ""
		}(),
	})
}

// NewAppError creates a new AppError
func NewAppError(errType ErrorType, code string, message string, severity ErrorSeverity) *AppError {
	return &AppError{
		Type:     errType,
		Code:     code,
		Message:  message,
		Severity: severity,
	}
}

// NewValidationError creates a validation error
func NewValidationError(code string, message string) *AppError {
	return NewAppError(ErrorTypeValidation, code, message, SeverityError)
}

// NewSystemError creates a system error
func NewSystemError(code string, message string, severity ErrorSeverity) *AppError {
	return NewAppError(ErrorTypeSystem, code, message, severity)
}

// NewNetworkError creates a network error
func NewNetworkError(code string, message string) *AppError {
	return NewAppError(ErrorTypeNetwork, code, message, SeverityError)
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(code string, message string) *AppError {
	return NewAppError(ErrorTypeConfiguration, code, message, SeverityError)
}

// NewRuntimeError creates a runtime error
func NewRuntimeError(code string, message string, severity ErrorSeverity) *AppError {
	return NewAppError(ErrorTypeRuntime, code, message, severity)
}

// NewDatabaseError creates a database error
func NewDatabaseError(code string, message string) *AppError {
	return NewAppError(ErrorTypeDatabase, code, message, SeverityError)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError converts an error to an AppError
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return NewAppError(ErrorTypeUnknown, "unknown_error", err.Error(), SeverityError).WithOriginalErr(err)
}

// ErrorResponse represents an error response for the API
type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   *AppError `json:"error"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err error) *ErrorResponse {
	var appErr *AppError
	if IsAppError(err) {
		appErr = err.(*AppError)
	} else {
		appErr = GetAppError(err)
	}

	return &ErrorResponse{
		Success: false,
		Error:   appErr,
	}
}
