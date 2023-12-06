package state

import "context"

// We need to know where to get logs from.
// For pending logs it'll be fetched from the buffer
// For completed logs they'll be fetched from the s3 bucket
type LogState byte

const (
	LogStateUnknown  LogState = 0
	LogStatePending  LogState = 1
	LogStateFinished LogState = 2
)

// Interface for state storage - we need to know if log is pending or finished
type Interface interface {
	Get(ctx context.Context, key string) (LogState, error)
	Put(ctx context.Context, key string, state LogState) error
}
