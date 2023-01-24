package triggers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_runLeaseChecker(t *testing.T) {
	t.Parallel()

	t.Run("should send true through leaseChan when lease identifier matches", func(t *testing.T) {
		t.Parallel()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
		defer cancel()

		mockLeaseBackend := NewMockLeaseBackend(mockCtrl)
		testClusterID := "testkube-api"
		testIdentifier := "test-host-1"
		mockLeaseBackend.EXPECT().TryAcquire(gomock.Any(), testIdentifier, testClusterID).Return(true, nil)

		s := &Service{
			identifier:         testIdentifier,
			clusterID:          testClusterID,
			leaseBackend:       mockLeaseBackend,
			leaseCheckInterval: 100 * time.Millisecond,
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

		ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
		defer cancel()

		mockLeaseBackend := NewMockLeaseBackend(mockCtrl)
		mockLeaseBackend.EXPECT().TryAcquire(gomock.Any(), "test-host-1", "testkube-api").Return(false, nil)

		s := &Service{
			identifier:         "test-host-1",
			clusterID:          "testkube-api",
			leaseBackend:       mockLeaseBackend,
			leaseCheckInterval: 100 * time.Millisecond,
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

		ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
		defer cancel()

		mockLeaseBackend1 := NewMockLeaseBackend(mockCtrl)
		mockLeaseBackend1.EXPECT().TryAcquire(gomock.Any(), "test-host-1", "testkube-api").Return(true, nil)
		mockLeaseBackend2 := NewMockLeaseBackend(mockCtrl)
		mockLeaseBackend2.EXPECT().TryAcquire(gomock.Any(), "test-host-2", "testkube-api").Return(false, nil)

		s1 := &Service{
			identifier:         "test-host-1",
			clusterID:          "testkube-api",
			leaseBackend:       mockLeaseBackend1,
			leaseCheckInterval: 100 * time.Millisecond,
			logger:             log.DefaultLogger,
		}

		leaseChan1 := make(chan bool)
		go s1.runLeaseChecker(ctx, leaseChan1)

		s2 := &Service{
			identifier:         "test-host-2",
			clusterID:          "testkube-api",
			leaseBackend:       mockLeaseBackend2,
			leaseCheckInterval: 100 * time.Millisecond,
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
