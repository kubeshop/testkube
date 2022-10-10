package triggers

import (
	"context"
	"time"
)

const (
	mongoCollectionTriggersLease = "triggers_lease"
	_id                          = "lease"
)

func (s *Service) runLeaseChecker(ctx context.Context, leaseChan chan<- bool) {
	ticker := time.NewTicker(s.leaseCheckInterval)
	s.logger.Debugf("trigger service: starting lease checker")

	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("trigger service: stopping lease checker component")
			return
		case <-ticker.C:
			lease, err := s.leaseBackend.CheckAndSet(ctx, s.identifier)
			if err != nil {
				s.logger.Errorf("error checking and setting lease: %v", err)
			}
			leaseChan <- lease.Identifier == s.identifier
		}
	}
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
