package errors

import (
	"fmt"
	"time"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     ErrorDetail `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Type    string `json:"type"`
}

// Error types
const (
	TypeValidation         = "validation_error"
	TypeAuthentication     = "authentication_error"
	TypeAuthorization      = "authorization_error"
	TypeNotFound           = "not_found"
	TypeConflict           = "conflict"
	TypeInternal           = "internal_error"
	TypeBadRequest         = "bad_request"
	TypeRateLimit          = "rate_limit_exceeded"
	TypeServiceUnavailable = "service_unavailable"
)

// Error codes
const (
	CodeInvalidInput      = "INVALID_INPUT"
	CodeMissingField      = "MISSING_FIELD"
	CodeInvalidFormat     = "INVALID_FORMAT"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeNotFound          = "NOT_FOUND"
	CodeAlreadyExists     = "ALREADY_EXISTS"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeDatabaseError     = "DATABASE_ERROR"
	CodeNetworkError      = "NETWORK_ERROR"
	CodeProviderError     = "PROVIDER_ERROR"
	CodePortUnavailable   = "PORT_UNAVAILABLE"
	CodeProxyStartFailed  = "PROXY_START_FAILED"
	CodeConfigError       = "CONFIG_ERROR"
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, err error) *ErrorResponse {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeInternalError,
			Message: message,
			Details: details,
			Type:    TypeInternal,
		},
		Timestamp: time.Now(),
	}
}

// NewValidationError creates a validation error response
func NewValidationError(message string, details string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeInvalidInput,
			Message: message,
			Details: details,
			Type:    TypeValidation,
		},
		Timestamp: time.Now(),
	}
}

// NewAuthenticationError creates an authentication error response
func NewAuthenticationError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeUnauthorized,
			Message: message,
			Type:    TypeAuthentication,
		},
		Timestamp: time.Now(),
	}
}

// NewAuthorizationError creates an authorization error response
func NewAuthorizationError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeForbidden,
			Message: message,
			Type:    TypeAuthorization,
		},
		Timestamp: time.Now(),
	}
}

// NewNotFoundError creates a not found error response
func NewNotFoundError(resource string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeNotFound,
			Message: fmt.Sprintf("%s not found", resource),
			Type:    TypeNotFound,
		},
		Timestamp: time.Now(),
	}
}

// NewConflictError creates a conflict error response
func NewConflictError(message string, details string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeAlreadyExists,
			Message: message,
			Details: details,
			Type:    TypeConflict,
		},
		Timestamp: time.Now(),
	}
}

// NewRateLimitError creates a rate limit error response
func NewRateLimitError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeRateLimitExceeded,
			Message: message,
			Type:    TypeRateLimit,
		},
		Timestamp: time.Now(),
	}
}

// NewProviderError creates a provider-specific error response
func NewProviderError(provider, message string, err error) *ErrorResponse {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeProviderError,
			Message: fmt.Sprintf("Provider %s error: %s", provider, message),
			Details: details,
			Type:    TypeInternal,
		},
		Timestamp: time.Now(),
	}
}

// NewPortUnavailableError creates a port unavailable error response
func NewPortUnavailableError(portRange string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodePortUnavailable,
			Message: fmt.Sprintf("No available ports in range %s", portRange),
			Type:    TypeServiceUnavailable,
		},
		Timestamp: time.Now(),
	}
}

// NewProxyStartError creates a proxy start error response
func NewProxyStartError(instanceID string, err error) *ErrorResponse {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeProxyStartFailed,
			Message: fmt.Sprintf("Failed to start proxy instance %s", instanceID),
			Details: details,
			Type:    TypeInternal,
		},
		Timestamp: time.Now(),
	}
}

// NewDatabaseError creates a database error response
func NewDatabaseError(operation string, err error) *ErrorResponse {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeDatabaseError,
			Message: fmt.Sprintf("Database %s failed", operation),
			Details: details,
			Type:    TypeInternal,
		},
		Timestamp: time.Now(),
	}
}

// NewConfigError creates a configuration error response
func NewConfigError(message string, err error) *ErrorResponse {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    CodeConfigError,
			Message: message,
			Details: details,
			Type:    TypeInternal,
		},
		Timestamp: time.Now(),
	}
}

// WithRequestID adds a request ID to the error response
func (e *ErrorResponse) WithRequestID(requestID string) *ErrorResponse {
	e.RequestID = requestID
	return e
}

// WithCode sets a custom error code
func (e *ErrorResponse) WithCode(code string) *ErrorResponse {
	e.Error.Code = code
	return e
}

// WithType sets a custom error type
func (e *ErrorResponse) WithType(errorType string) *ErrorResponse {
	e.Error.Type = errorType
	return e
}

// AppError represents an application-specific error
type AppError struct {
	Code    string
	Message string
	Cause   error
	Type    string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(code, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Type:    TypeInternal,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error chain
func GetAppError(err error) (*AppError, bool) {
	var appErr *AppError
	for err != nil {
		if ae, ok := err.(*AppError); ok {
			appErr = ae
			break
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return appErr, appErr != nil
}
