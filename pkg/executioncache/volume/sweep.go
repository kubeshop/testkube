package volume

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Sweeper deletes entries the object store has already forgotten about.
//
// The object store expires the pointers on its own, through the bucket lifecycle rule
// the cache prefix already carries, so the index needs nothing here. What it cannot
// expire is what the pointers point at: a volume has no lifecycle rules, and unlike a
// bucket it has a fixed capacity, so without this it fills once and then every save
// fails forever.
//
// It works on whole inbox directories rather than on entries, and on mtime rather than
// on any index, because there is deliberately no index to consult. The pointers live in
// the object store under keys this process cannot enumerate cheaply, and one that could
// would still be racing the lifecycle rule deleting them. An inbox belongs to one
// execution, which is finished long before the retention window, so the directory's own
// age is sound and self-contained.
//
// Retention must be at least the object store's cache expiration. The ordering matters
// one way only: an entry outliving its pointer wastes space until the next sweep, where
// a pointer outliving its entry is a restore that reports a miss it cannot explain.
type Sweeper struct {
	// Root is the volume, mounted where this process can reach it.
	Root string
	// Retention is how long after an execution last wrote to its inbox the entries in
	// it survive.
	Retention time.Duration
	// Interval is how often to sweep.
	Interval time.Duration
	// Now is the clock, so a test does not have to wait out a retention window.
	Now func() time.Time
	// OnError reports a sweep that could not finish. A sweep is maintenance, so a
	// failure is logged and retried at the next interval rather than returned.
	OnError func(error)
}

// Run sweeps until the context is cancelled.
//
// The first sweep happens immediately: an agent that has just started may be taking
// over from one that was killed mid-window, and waiting a full interval to find a full
// volume helps nobody.
func (s *Sweeper) Run(ctx context.Context) error {
	interval := s.Interval
	if interval <= 0 {
		interval = time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := s.Sweep(ctx); err != nil && s.OnError != nil {
			s.OnError(err)
		}
		select {
		case <-ctx.Done():
			// Returning nil rather than ctx.Err(): the coordinator cancels this on
			// losing leadership, which is an ordinary handover and not a failure.
			return nil
		case <-ticker.C:
		}
	}
}

// Sweep removes every inbox older than the retention window, once.
//
// It checks the context between inboxes rather than only at the start, because the
// leader coordinator waits for this to return before handing leadership over, and a
// volume holding thousands of inboxes would otherwise stall that for the length of a
// full walk.
func (s *Sweeper) Sweep(ctx context.Context) error {
	if s.Root == "" || s.Retention <= 0 {
		return nil
	}

	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	cutoff := now().Add(-s.Retention)

	inboxes, err := os.ReadDir(filepath.Join(s.Root, InboxDir))
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing has been cached yet, which is the normal first state rather than
			// a fault - the same way a cold bucket is.
			return nil
		}
		return err
	}

	for _, entry := range inboxes {
		if ctx.Err() != nil {
			return nil
		}
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(s.Root, InboxDir, entry.Name())

		// The inbox's own mtime moves whenever an entry is added to or removed from it,
		// so it tracks the execution's last write without having to walk inside.
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.ModTime().After(cutoff) {
			continue
		}

		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// PointerLifetime reports how long a cache pointer can still be served, and whether
// anything expires it at all.
//
// The sweep has to outlast it. An entry removed while the pointer naming it is still
// stored turns every hit on that key into a miss the restore cannot explain, and the
// key is immutable, so no later run can replace it until the object itself expires.
// Keeping an entry nothing points at only wastes space until the next sweep, so that
// is the direction to err in.
//
// Two lifecycle rules can match one cache object - the cache-prefixed one and the
// bucket-wide one, which is deliberately left unfiltered and so covers cache objects
// too - and where both do, the object store applies the earlier. Either count being 0
// means that rule is disabled. Both being 0 means pointers are kept indefinitely, which
// no retention can outlast; that is reported rather than answered with a duration,
// because returning 0 would read as "already expired" and silently pass any retention.
func PointerLifetime(cacheExpirationDays, bucketExpirationDays int) (time.Duration, bool) {
	const day = 24 * time.Hour
	switch {
	case cacheExpirationDays > 0 && bucketExpirationDays > 0:
		return time.Duration(min(cacheExpirationDays, bucketExpirationDays)) * day, true
	case cacheExpirationDays > 0:
		return time.Duration(cacheExpirationDays) * day, true
	case bucketExpirationDays > 0:
		return time.Duration(bucketExpirationDays) * day, true
	default:
		return 0, false
	}
}
