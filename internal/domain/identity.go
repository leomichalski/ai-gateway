package domain

// Identity represents the authenticated caller of the AI Gateway.
type Identity struct {
	TenantID   string
	Tier       string
	APIKeyHash string
}
