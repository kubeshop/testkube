package grpcutils

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorCode extracts the gRPC status code from an error, or Unknown when the
// error does not carry one.
func ErrorCode(err error) codes.Code {
	if err == nil {
		return codes.Unknown
	}
	if e, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return e.GRPCStatus().Code()
	}
	return codes.Unknown
}

// IsTransient reports whether an error is worth retrying: the server was
// unreachable, busy, or did not answer in time.
//
// Anything else - NotFound, PermissionDenied, InvalidArgument - will fail the
// same way on the next attempt, so a caller that retries indefinitely on those
// never makes progress.
func IsTransient(err error) bool {
	return IsRetryableCode(ErrorCode(err))
}

// IsRetryableCode reports whether a gRPC status code is worth retrying.
func IsRetryableCode(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	}
	return false
}
