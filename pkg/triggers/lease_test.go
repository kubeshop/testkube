package triggers

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestService_runLeaseChecker(t *testing.T) {
	t.Parallel()

	t.Run("should send true through leaseChan when lease identifier matches", func(t *testing.T) {
		t.Parallel()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 1300*time.Millisecond)
		defer cancel()

		mockLeaseBackend := NewMockLeaseBackend(mockCtrl)
		testIdentifier := "test-host-1"
		testLease := Lease{
			Identifier: testIdentifier,
			ClusterID:  "testkube",
			AcquiredAt: time.Now(),
			RenewedAt:  time.Now(),
		}
		mockLeaseBackend.EXPECT().CheckAndSet(gomock.Any(), testIdentifier).Return(&testLease, nil)

		s := &Service{
			identifier:         testIdentifier,
			leaseBackend:       mockLeaseBackend,
			leaseCheckInterval: 1 * time.Second,
			logger:             log.DefaultLogger,
		}

		leaseChan := make(chan bool)
		go s.runLeaseChecker(ctx, leaseChan)

		select {
		case <-ctx.Done():
			t.Errorf("did not receive lease response in expected timeframe")
		case lease := <-leaseChan:
			assert.True(t, lease, "instance should acquire lease")
		}
	})

	t.Run("should send false through leaseChan when lease identifier do not match", func(t *testing.T) {
		t.Parallel()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 1300*time.Millisecond)
		defer cancel()

		mockLeaseBackend := NewMockLeaseBackend(mockCtrl)
		testLease := Lease{
			Identifier: "test-host-2",
			ClusterID:  "testkube",
			AcquiredAt: time.Now(),
			RenewedAt:  time.Now(),
		}
		mockLeaseBackend.EXPECT().CheckAndSet(gomock.Any(), "test-host-1").Return(&testLease, nil)

		s := &Service{
			identifier:         "test-host-1",
			leaseBackend:       mockLeaseBackend,
			leaseCheckInterval: 1 * time.Second,
			logger:             log.DefaultLogger,
		}

		leaseChan := make(chan bool)
		go s.runLeaseChecker(ctx, leaseChan)

		select {
		case <-ctx.Done():
			t.Errorf("did not receive lease response in expected timeframe")
		case lease := <-leaseChan:
			assert.False(t, lease, "instance should not acquire lease")
		}
	})
}

func TestService_runLeaseChecker_multipleInstances(t *testing.T) {
	t.Parallel()

	t.Run("only one instance should acquire lease successfully", func(t *testing.T) {
		t.Parallel()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 1300*time.Millisecond)
		defer cancel()

		mockLeaseBackend1 := NewMockLeaseBackend(mockCtrl)
		mockLeaseBackend2 := NewMockLeaseBackend(mockCtrl)
		testLease := Lease{
			Identifier: "test-host-1",
			ClusterID:  "testkube",
			AcquiredAt: time.Now(),
			RenewedAt:  time.Now(),
		}
		mockLeaseBackend1.EXPECT().CheckAndSet(gomock.Any(), "test-host-1").Return(&testLease, nil)
		mockLeaseBackend2.EXPECT().CheckAndSet(gomock.Any(), "test-host-2").Return(&testLease, nil)

		s1 := &Service{
			identifier:         "test-host-1",
			leaseBackend:       mockLeaseBackend1,
			leaseCheckInterval: 1 * time.Second,
			logger:             log.DefaultLogger,
		}

		leaseChan1 := make(chan bool)
		go s1.runLeaseChecker(ctx, leaseChan1)

		s2 := &Service{
			identifier:         "test-host-2",
			leaseBackend:       mockLeaseBackend2,
			leaseCheckInterval: 1 * time.Second,
			logger:             log.DefaultLogger,
		}

		leaseChan2 := make(chan bool)
		go s2.runLeaseChecker(ctx, leaseChan2)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				t.Errorf("did not receive lease from test-host-1")
			case lease := <-leaseChan1:
				assert.True(t, lease, "first instance should acquire lease")
			}
		}()

		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				t.Errorf("did not receive lease from test-host-1")
			case lease := <-leaseChan2:
				assert.False(t, lease, "second instance should not acquire lease")
			}
		}()

		wg.Wait()
	})
}
