package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// The sweep decides what to keep from the leases it finds on the volume, and a leader
// that has just taken over has written none of them. If the first pass cannot say which
// executions are running, every live inbox looks abandoned - and the sweep that follows
// unlinks the directory a pod is writing through its subPath into, leaving it writing to
// an inode nothing can reach and then publishing a pointer naming a path that is gone.
//
// So the failure has to hold the sweep rather than pass through it.
func TestTheSweepDoesNotStartUntilTheLiveSetIsKnown(t *testing.T) {
	worker := executionworkertypes.NewMockWorker(gomock.NewController(t))
	worker.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("the api server is unreachable")).
		AnyTimes()

	// Cancelled while it is still waiting, which is what a handover looks like: it
	// reports that the live set was never established, and the caller does not sweep.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	assert.False(t, awaitStepCacheLeases(ctx, worker, t.TempDir()),
		"nothing may be swept while it is unknown which executions are running")
}

// And it keeps trying rather than giving up, because giving up would leave the shared
// volume filling with no way back short of restarting the agent.
func TestTheSweepStartsOnceTheLiveSetCanBeRead(t *testing.T) {
	previous := stepCacheLeaseStartupRetry
	stepCacheLeaseStartupRetry = time.Millisecond
	defer func() { stepCacheLeaseStartupRetry = previous }()

	root := t.TempDir()
	inbox := volume.InboxFor("exec-1")
	require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.FromSlash(inbox)), volume.SharedDirMode))

	worker := executionworkertypes.NewMockWorker(gomock.NewController(t))
	gomock.InOrder(
		worker.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("the api server is unreachable")),
		worker.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return([]executionworkertypes.ListResultItem{}, nil),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	assert.True(t, awaitStepCacheLeases(ctx, worker, root),
		"a listing that recovers has to let the sweep start")
}
