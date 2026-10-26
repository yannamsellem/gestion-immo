package domain

import "errors"

var (
	ErrPaymentNotFound     = errors.New("payment: payment not found")
	ErrValidation          = errors.New("payment: validation error")
	ErrOpenBankingDisabled = errors.New("payment: open banking DSP2 synchronization is disabled")
)
