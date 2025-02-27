package controlplaneclient

import "errors"

var (
	ErrNotSupported = errors.New("operation not supported in this version")
)
