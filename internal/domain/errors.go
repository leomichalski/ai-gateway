package domain

import (
	"errors"
	"fmt"
)

// =====================================================================
// 1. Core Sentinel Errors (The "What went wrong")
// =====================================================================
// These are used for simple, direct comparisons using errors.Is()
// e.g., if errors.Is(err, domain.ErrClientCanceled)

var (
	// Client / Network related
	ErrClientCanceled = errors.New("operation aborted by client context cancellation")
	ErrGatewayTimeout = errors.New("gateway operation timed out")

	// Upstream Provider related
	ErrProviderFailure       = errors.New("upstream AI provider failed")
	ErrEmptyProviderResponse = errors.New("provider returned an empty response")
	ErrModelInitialization   = errors.New("failed to initialize AI model")

	// Security & Authorization
	// ErrUnauthorized      = errors.New("unauthorized access")
	// ErrRateLimitExceeded = errors.New("rate limit or quota exceeded")
	ErrInternalSecret    = errors.New("failed to retrieve internal secrets")

	// Business / Validation
	ErrInvalidRequest = errors.New("invalid gateway request parameters")

	// Secrets / Config
	ErrSecretNotFound = errors.New("provider secret key not found")
	ErrSecretNotSet   = errors.New("provider secret key is empty or not configured")

	// Streaming
	ErrStreamInterrupted = errors.New("streaming connection interrupted by upstream provider")
)

// =====================================================================
// 2. Custom Error Type (The "Why it went wrong")
// =====================================================================
// ErrorType defines the category of the error to help the Delivery layer
// map it to the correct protocol response (HTTP/gRPC) without coupling.
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	ErrorTypeAuth       ErrorType = "AUTH_ERROR"
	ErrorTypeUpstream   ErrorType = "UPSTREAM_ERROR"
	ErrorTypeInternal   ErrorType = "INTERNAL_ERROR"
	ErrorTypeTimeout    ErrorType = "TIMEOUT_ERROR"
)

// GatewayError is a custom error struct that implements the Go error interface.
// It allows Adapters and Use Cases to wrap underlying errors with rich context.
type GatewayError struct {
	Type    ErrorType
	Message string
	Err     error  // The underlying error (e.g., from adk-go or net/http)
}

// Error implements the standard Go error interface.
func (e *GatewayError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap allows Go 1.13+ errors.Is() and errors.As() to traverse the error chain.
func (e *GatewayError) Unwrap() error {
	return e.Err
}

// =====================================================================
// 3. Helper Constructors
// =====================================================================

// NewGatewayError creates a richly typed domain error.
func NewGatewayError(errType ErrorType, message string, underlying error) *GatewayError {
	return &GatewayError{
		Type:    errType,
		Message: message,
		Err:     underlying,
	}
}
