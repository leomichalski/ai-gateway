package outbound

import "context"

type PermissionEnforcer interface {
	// CheckAccess returns true if the subject (tenant) can perform the action on the object (model)
	CheckAccess(ctx context.Context, subject, object, action string) (bool, error)
}
