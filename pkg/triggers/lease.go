package triggers

import (
	"context"
	"time"
)

func (s *Service) runLeaseChecker(ctx context.Context, leaseChan chan<- bool) {
	ticker := time.NewTicker(s.leaseCheckInterval)
	s.logger.Debugf("trigger service: starting lease checker")

	s.logger.Info("trigger service: waiting for lease")

	// check for lease immediately on startup instead of waiting for first ticker iteration
	s.leaseCheckerIteration(ctx, leaseChan)

	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("trigger service: stopping lease checker component")
			return
		case <-ticker.C:
			s.leaseCheckerIteration(ctx, leaseChan)
		}
	}
}

func (s *Service) leaseCheckerIteration(ctx context.Context, leaseChan chan<- bool) {
	leased, err := s.leaseBackend.TryAcquire(ctx, s.identifier, s.clusterID)
	if err != nil {
		s.logger.Errorf("error checking and setting lease: %v", err)
	}
	leaseChan <- leased
}

type Lease struct {
	Identifier string    `bson:"identifier"`
	ClusterID  string    `bson:"cluster_id"`
	AcquiredAt time.Time `bson:"acquired_at"`
	RenewedAt  time.Time `bson:"renewed_at"`
}

func NewLease(identifier, clusterID string) *Lease {
	return &Lease{
		Identifier: identifier,
		ClusterID:  clusterID,
		AcquiredAt: time.Now(),
		RenewedAt:  time.Now(),
	}
}
