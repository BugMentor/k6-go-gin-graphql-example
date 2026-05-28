package model

import "errors"

var (
	ErrInvalidEmail          = errors.New("invalid email")
	ErrEmptyFullName         = errors.New("full name cannot be empty")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrPositiveAmountRequired = errors.New("amount must be positive")
	ErrRefundNotAllowed      = errors.New("only SUCCESS payments can be refunded")
	ErrNotFound              = errors.New("entity not found")
	ErrVersionConflict       = errors.New("version conflict: concurrent update")
)
