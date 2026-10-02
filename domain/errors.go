package domain

import "errors"

var (
	ErrStorageUnavailable = errors.New("storage unavailable")
	ErrNotFound           = errors.New("not found")
	ErrInvalidTransition  = errors.New("invalid state transition")
)
