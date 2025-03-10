package common

import "errors"

const (
	ModeStandalone = "standalone"
	ModeAgent      = "agent"
)

var ErrNotSupported = errors.New("Feature is not supported in standalone mode")
