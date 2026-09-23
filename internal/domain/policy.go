package domain

import "errors"

var (
	ErrForbidden = errors.New("requested object is not allowed for this tenant")
	ErrEnforcerFailed  = errors.New("internal error while verifying permissions")
)
