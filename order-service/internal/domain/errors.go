package domain

import "errors"

var (
	ErrInvalidAmount      = errors.New("amount must be greater than 0")
	ErrOrderNotFound      = errors.New("order not found")
	ErrCannotCancelPaid   = errors.New("paid orders cannot be cancelled")
	ErrCannotCancelStatus = errors.New("only pending orders can be cancelled")
	ErrPaymentUnavailable = errors.New("payment service unavailable")
)
