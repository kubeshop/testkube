package grpc

import "strings"

// The runner sends one of these tokens to the control plane with a decline. The
// values are stable, because filters and telemetry use them. The error text goes
// in the message.
const (
	StartReasonImagePullFailed   = "image-pull-failed"
	StartReasonDefinitionInvalid = "definition-invalid"
	StartReasonResourceFailed    = "resource-create-failed"
	StartReasonJobCreateFailed   = "job-create-failed"
	StartReasonUnknown           = "start-failed"
)

// startErrorReason maps a start error to a reason token. The worker wraps each
// start error in a fixed text per phase, and that text selects the token. The
// checks run in this order because the processing text contains the image error,
// and the deploy text contains the secret, config map, and volume claim errors.
// The function returns the first token that matches.
func startErrorReason(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "resolving image error"):
		return StartReasonImagePullFailed
	case strings.Contains(msg, "failed to process test workflow"):
		return StartReasonDefinitionInvalid
	case strings.Contains(msg, "failed to deploy secrets"),
		strings.Contains(msg, "failed to deploy config maps"),
		strings.Contains(msg, "failed to deploy pvcs"):
		return StartReasonResourceFailed
	case strings.Contains(msg, "failed to deploy"),
		strings.Contains(msg, "admission webhook"),
		strings.Contains(msg, "is invalid"):
		return StartReasonJobCreateFailed
	default:
		return StartReasonUnknown
	}
}
