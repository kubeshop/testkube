package k8s

import (
	"context"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
)

// K8sLeaseBackend implements lease acquisition using Kubernetes Lease objects.
// It allows multiple API server instances to coordinate without an external DB.
//
// Semantics:
// - If the Lease does not exist, it is created and acquired by this instance
// - If the Lease exists and is held by this instance, it is renewed
// - If the Lease exists and is held by another instance but expired, it is taken over
// - Otherwise, acquisition fails (returns leased=false)
type K8sLeaseBackend struct {
	client        kubernetes.Interface
	name          string
	namespace     string
	leaseDuration time.Duration
}

type Option func(*K8sLeaseBackend)

// WithLeaseDuration overrides the default max lease duration.
func WithLeaseDuration(d time.Duration) Option {
	return func(b *K8sLeaseBackend) { b.leaseDuration = d }
}

// NewK8sLeaseBackend creates a K8s-backed lease backend using coordination.k8s.io Leases in the given namespace.
func NewK8sLeaseBackend(client kubernetes.Interface, name, namespace string, opts ...Option) *K8sLeaseBackend {
	b := &K8sLeaseBackend{
		client:        client,
		name:          name,
		namespace:     namespace,
		leaseDuration: leasebackend.DefaultMaxLeaseDuration,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// TryAcquire attempts to acquire or renew the Lease for the provided clusterID with holder id.
func (b *K8sLeaseBackend) TryAcquire(ctx context.Context, id, clusterID string) (bool, error) { //nolint:revive,unused
	leaseName := b.leaseName(clusterID)
	leases := b.client.CoordinationV1().Leases(b.namespace)

	now := metav1.MicroTime{Time: time.Now()}
	leaseDurationSeconds := int32(b.leaseDuration.Seconds())

	// Get current lease
	l, err := leases.Get(ctx, leaseName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Create and acquire new lease
		lease := &coordv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: b.namespace,
			},
			Spec: coordv1.LeaseSpec{
				HolderIdentity:       &id,
				AcquireTime:          &now,
				RenewTime:            &now,
				LeaseDurationSeconds: &leaseDurationSeconds,
			},
		}
		if _, err := leases.Create(ctx, lease, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) || apierrors.IsConflict(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// Renew if we already hold it
	if l.Spec.HolderIdentity != nil && *l.Spec.HolderIdentity == id {
		l.Spec.RenewTime = &now
		if l.Spec.LeaseDurationSeconds == nil {
			l.Spec.LeaseDurationSeconds = &leaseDurationSeconds
		}
		if _, err := leases.Update(ctx, l, metav1.UpdateOptions{}); err != nil {
			if apierrors.IsConflict(err) {
				// Someone updated concurrently; try again on next tick.
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	// Check expiry
	expired := true
	if l.Spec.RenewTime != nil && l.Spec.LeaseDurationSeconds != nil {
		expired = time.Since(l.Spec.RenewTime.Time) > time.Duration(*l.Spec.LeaseDurationSeconds)*time.Second
	}
	if !expired {
		return false, nil
	}

	// Take over expired lease
	l.Spec.HolderIdentity = &id
	l.Spec.AcquireTime = &now
	l.Spec.RenewTime = &now
	if l.Spec.LeaseDurationSeconds == nil {
		l.Spec.LeaseDurationSeconds = &leaseDurationSeconds
	}

	if _, err := leases.Update(ctx, l, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsConflict(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b *K8sLeaseBackend) leaseName(clusterID string) string {
	return b.name + "-" + clusterID
}
