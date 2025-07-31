/**
 * Error type representing the category of an error
 */
export type ErrorType = 
  | 'validation'    // Validation errors (invalid input, etc.)
  | 'system'        // System errors (file system, permissions, etc.)
  | 'network'       // Network errors (download failures, etc.)
  | 'configuration' // Configuration errors (invalid config, etc.)
  | 'runtime'       // Runtime errors (service failures, etc.)
  | 'database'      // Database errors
  | 'api'           // API errors
  | 'unknown';      // Unknown errors

/**
 * Error severity representing the severity of an error
 */
export type ErrorSeverity = 
  | 'fatal'    // Fatal errors that require immediate attention
  | 'critical' // Critical errors that may impact functionality
  | 'error'    // Standard errors
  | 'warning'  // Warnings that don't prevent operation
  | 'info';    // Informational messages

/**
 * Application error structure
 */
export interface AppError {
  type: ErrorType;
  code: string;
  message: string;
  severity: ErrorSeverity;
  details?: any;
  original_error?: string;
}

/**
 * Error response from the API
 */
export interface ErrorResponse {
  success: boolean;
  error: AppError;
}

/**
 * Creates a user-friendly error message based on an AppError
 */
export const getUserFriendlyErrorMessage = (error: AppError): string => {
  // Return the error message directly if it's already user-friendly
  if (error.type === 'validation' || error.type === 'configuration') {
    return error.message;
  }

  // Create user-friendly messages based on error type
  switch (error.type) {
    case 'network':
      return 'Network error: Unable to connect to the server. Please check your internet connection and try again.';
    case 'system':
      return 'System error: The server encountered an issue. Please try again later.';
    case 'runtime':
      return 'Application error: Something went wrong while processing your request. Please try again.';
    case 'database':
      return 'Database error: Unable to access data. Please try again later.';
    case 'api':
      return `API error: ${error.message}`;
    default:
      return 'An unexpected error occurred. Please try again later.';
  }
};

/**
 * Creates a user-friendly error message with technical details
 */
export const getDetailedErrorMessage = (error: AppError): string => {
  return `${getUserFriendlyErrorMessage(error)}\n\nTechnical details: [${error.type}:${error.code}] ${error.message}`;
};

/**
 * Determines if an error should be displayed to the user
 */
export const shouldDisplayError = (error: AppError): boolean => {
  // Always display validation and configuration errors
  if (error.type === 'validation' || error.type === 'configuration') {
    return true;
  }

  // Display errors based on severity
  return ['fatal', 'critical', 'error'].includes(error.severity);
};

/**
 * Gets the appropriate CSS class for an error based on severity
 */
export const getErrorAlertClass = (error: AppError): string => {
  switch (error.severity) {
    case 'fatal':
    case 'critical':
      return 'alert-error';
    case 'error':
      return 'alert-error';
    case 'warning':
      return 'alert-warning';
    case 'info':
      return 'alert-info';
    default:
      return 'alert-error';
  }
};

/**
 * Creates an error object from any caught exception
 */
export const createErrorFromException = (error: any): AppError => {
  if (error && typeof error === 'object' && 'type' in error && 'code' in error && 'message' in error) {
    return error as AppError;
  }

  return {
    type: 'unknown',
    code: 'uncaught_exception',
    message: error?.message || 'An unknown error occurred',
    severity: 'error',
    details: error
  };
};