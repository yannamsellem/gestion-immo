package domain

import "errors"

var (
	ErrUserNotFound              = errors.New("auth: user not found")
	ErrEmailAlreadyExists        = errors.New("auth: email already registered")
	ErrInvalidCredentials        = errors.New("auth: invalid email or password")
	ErrAccountNotApproved        = errors.New("auth: account is not approved by SUPER_ADMIN")
	ErrForbiddenSuperAdminOnly   = errors.New("auth: action restricted to SUPER_ADMIN")
	ErrInvalidToken              = errors.New("auth: invalid or expired token")
	ErrRefreshTokenReuseDetected = errors.New("auth: refresh token reuse detected, token family revoked")
	ErrValidation                = errors.New("auth: validation error")
)
