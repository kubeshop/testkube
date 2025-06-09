package leasebackend

import (
	context "context"
	"time"
)

const (
	DefaultMaxLeaseDuration = 1 * time.Minute
)

// LeaseBackend does a check and set operation on the Lease object in the defined data source
//
//go:generate mockgen -destination=./mock_lease_backend.go -package=triggers "github.com/kubeshop/testkube/pkg/triggers" LeaseBackend
type LeaseBackend interface {
	// TryAcquire tries to acquire lease from underlying datastore
	TryAcquire(ctx context.Context, id, clusterID string) (leased bool, err error)
}
