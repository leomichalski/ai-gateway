package domain

import "errors"

var (
	ErrIdentityNotFound = errors.New("identity not found")
)

// Identity represents the authenticated caller of the AI Gateway.
type Identity struct {
	TenantID   string
	Tier       string
	APIKeyHash string
}
