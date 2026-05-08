package mongo

import "time"

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
