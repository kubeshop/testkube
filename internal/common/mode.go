package common

import "errors"

const (
	ModeStandalone    = "standalone"
	ModeAgent         = "agent"
	ModeListenerAgent = "listener-agent"
	ModeGitOpsAgent   = "gitops-agent"
)

var ErrNotSupported = errors.New("Feature is not supported in standalone mode")
