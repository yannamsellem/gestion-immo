package domain

import "errors"

var (
	ErrLegalEntityNotFound = errors.New("portfolio: legal entity not found")
	ErrPropertyNotFound    = errors.New("portfolio: property not found")
	ErrLotNotFound         = errors.New("portfolio: lot not found")
	ErrShareholderNotFound = errors.New("portfolio: shareholder not found")
	ErrDuplicateSIRET      = errors.New("portfolio: legal entity with this SIRET already exists")
	ErrValidation          = errors.New("portfolio: validation error")
)
