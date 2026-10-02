package volume

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	// LeaseTTL is how long a lease keeps an inbox, counted from its last refresh.
	//
	// An execution holds its inbox for as long as it runs, however quiet it is, and
	// mtime cannot see that - see LeaseName. Zero disables the check, which sweeps on
	// mtime alone and may unlink a directory a running pod still writes to.
	//
	// Comfortably above the refresh interval, so that one missed refresh does not
	// expose a live inbox.
	LeaseTTL time.Duration
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

		// A live execution holds its inbox whether or not it has written to it lately,
		// and only the lease can say so. Checked before mtime, because an execution
		// that has cached nothing yet is exactly the one mtime judges most harshly.
		if s.LeaseTTL > 0 {
			switch lease, err := os.Stat(filepath.Join(dir, LeaseName)); {
			case err == nil:
				// Age, not "before now plus the window": a negative age is a timestamp
				// in the future, and a future one stays in the future, so it would
				// keep this inbox for good. The step's own command holds the inbox
				// mount - the save stage is pure and merges into it - so a workflow
				// can write .lease itself and date it a century ahead, and the shared
				// volume would never reclaim that directory again.
				//
				// A little slack below zero, because the agent and whatever serves the
				// volume keep their own clocks and a lease written moments ago may
				// read as marginally ahead.
				if age := now().Sub(lease.ModTime()); age >= -maxClockSkew && age < s.LeaseTTL {
					continue
				}
			case !os.IsNotExist(err):
				return err
			}
		}

		// The inbox's own mtime moves whenever an entry is added to or removed from it,
		// so it tracks the execution's last write without having to walk inside.
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		// A future mtime is kept from pinning the inbox the same way, and for the same
		// reason: the step can touch its own inbox directory. Beyond the skew window it
		// is not a clock difference, so it is treated as expired rather than as
		// infinitely recent. A running execution is still safe - the lease above is
		// what holds a live inbox, and the agent writes that one.
		modified := info.ModTime()
		if modified.After(cutoff) && !modified.After(now().Add(maxClockSkew)) {
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

// LeaseName is the file inside an inbox whose mtime says an execution still holds it.
//
// The inbox's own mtime cannot answer that. It is made before the pod starts and only
// moves when something is staged directly inside it, so an execution that caches
// nothing for a while - or runs one long step - looks untouched however alive it is.
// Sweeping it then unlinks a directory a running pod still has mounted through its
// subPath: the writes go to an inode nothing can reach, and the pointer that execution
// publishes names a path that no longer exists, which is a key that misses until its
// object expires.
// maxClockSkew is how far ahead of this process a timestamp on the volume may be and
// still be believed.
//
// The agent and whatever serves the volume keep their own clocks, so a lease written
// moments ago can read as marginally ahead. Anything further is not a clock difference:
// the step's own command holds the inbox mount, so a workflow can date a timestamp a
// century ahead, and without a bound that inbox would never be reclaimed.
const maxClockSkew = 5 * time.Minute

const LeaseName = ".lease"

// TouchLease marks an inbox as still in use, and is the other half of Sweeper.LeaseTTL.
//
// It never creates the inbox, only the lease inside one that is already there, so a
// refresh arriving after a sweep cannot resurrect a directory nothing is mounted at.
//
// Deliberately time-based rather than a flag the agent sets and clears. A flag that was
// never cleared - a crash, a lost watch, a replica that went away - would pin an inbox
// for good and fill the volume with no way to reclaim it. A lease that stops being
// refreshed goes stale on its own, and the sweep carries on.
func TouchLease(mountPath, inboxName string) error {
	dir, err := inboxPath(mountPath, inboxName)
	if err != nil {
		return err
	}

	name := filepath.Join(dir, LeaseName)
	now := time.Now()
	if err := os.Chtimes(name, now, now); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// O_CREATE without MkdirAll: a missing parent is a swept or never-made inbox, and
	// the error says so rather than building one nothing will ever read.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	return f.Close()
}

// inboxPath resolves an inbox name under the mount, refusing anything that is not
// exactly inbox/<segment> so that a lease cannot be written outside one.
func inboxPath(mountPath, inboxName string) (string, error) {
	if mountPath == "" {
		return "", errors.New("no shared cache volume is configured")
	}
	parts := strings.Split(path.Clean(inboxName), "/")
	if len(parts) != 2 || parts[0] != InboxDir || parts[1] == "" || parts[1] == "." || parts[1] == ".." {
		return "", fmt.Errorf("%q is not an inbox name", inboxName)
	}
	return filepath.Join(mountPath, parts[0], parts[1]), nil
}
