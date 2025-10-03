package common

import "errors"

const (
	ModeStandalone = "standalone"
	ModeAgent      = "agent"
)

var ErrNotSupported = errors.New("feature is not supported in standalone mode")
