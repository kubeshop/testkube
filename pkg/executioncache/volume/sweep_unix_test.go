//go:build unix

package volume

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lease sits inside a directory the step's own command can write, so a workflow can
// replace it with a symlink pointing at itself: every Stat of it then answers ELOOP.
//
// Returning that error stopped the sweep at this inbox on every pass, so nothing after
// it was ever reclaimed and the shared volume filled - one workflow able to wedge the
// cache for every execution in the cluster. A lease that cannot be read is simply not
// believed, so the inbox expires by mtime the ordinary way and the sweep carries on.
//
// Unix-only because creating the self-referential link is the whole setup.
func TestSweepContinuesWhenALeaseCannotBeRead(t *testing.T) {
	root := t.TempDir()
	wedged := inboxAged(t, root, "exec-wedged", 48*time.Hour)
	other := inboxAged(t, root, "exec-other", 48*time.Hour)
	require.NoError(t, os.Symlink(LeaseName, filepath.Join(wedged, LeaseName)))

	// The symlink is the last thing written, so put the directory back out of the
	// retention window.
	age(t, wedged)

	s := &Sweeper{Root: root, Retention: time.Hour, LeaseTTL: time.Hour}
	err := s.Sweep(context.Background())

	_, otherErr := os.Stat(other)
	assert.True(t, os.IsNotExist(otherErr), "an unreadable lease must not strand the inboxes after it")
	_, wedgedErr := os.Stat(wedged)
	assert.True(t, os.IsNotExist(wedgedErr), "and the inbox it could not vouch for expires the ordinary way")

	require.Error(t, err, "the operator still has to hear that a lease could not be read")
	assert.Contains(t, err.Error(), "exec-wedged")
}
