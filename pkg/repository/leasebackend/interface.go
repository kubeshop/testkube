package leasebackend

import (
	context "context"
	"time"
)

const (
	DefaultMaxLeaseDuration = 1 * time.Minute
)

// Repository does a check and set operation on the Lease object in the defined data source
//
//go:generate mockgen -destination=./mock_repository.go -package=leasebackend "github.com/kubeshop/testkube/pkg/repository/leasebackend" Repository
type Repository interface {
	// TryAcquire tries to acquire lease from underlying datastore
	TryAcquire(ctx context.Context, id, clusterID string) (leased bool, err error)
}
