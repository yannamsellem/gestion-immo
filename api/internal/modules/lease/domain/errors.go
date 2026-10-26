package domain

import "errors"

var (
	ErrTenantNotFound   = errors.New("lease: tenant not found")
	ErrLeaseNotFound    = errors.New("lease: lease not found")
	ErrRentCallNotFound = errors.New("lease: rent call not found")
	ErrDuplicatePeriod  = errors.New("lease: rent call already exists for lease and period")
	ErrValidation       = errors.New("lease: validation error")
)
