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
	// Protected reports an inbox this process knows is live, whatever its lease and
	// mtime say. Optional; nil protects nothing.
	//
	// The lease files are how one agent tells *another* that an inbox is live, and
	// across deployments sharing a volume they are the only way. Within one agent they
	// are an indirection that can fail: the step holds the inbox mount, so .lease is a
	// file this process may be unable to write, and a renewal that fails leaves a live
	// inbox looking abandoned to this agent's own sweep. What the agent knows directly
	// it should not have to read back off the disk to believe.
	Protected func(inbox string) bool
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

	// Collected rather than returned at the first failure, so the sweep reaches every
	// inbox and the caller still hears about the ones it could not take.
	var failed []string

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
		// entry.Info rather than entry.IsDir, and read once here because the mtime
		// below wants it too.
		//
		// The two agree: os resolves a kind readdir did not know with an lstat before
		// handing the DirEntry over (file_unix.go, newUnixDirent), so IsDir is not the
		// hazard on a network filesystem it looks like. This is one source for both
		// questions rather than a guard against that.
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				// Swept by another agent between the listing and here.
				continue
			}
			failed = append(failed, fmt.Sprintf("%s: reading it: %s", entry.Name(), err))
			continue
		}
		if !info.IsDir() {
			continue
		}

		// Asked before the lease and before the mtime, because it is the one answer
		// that cannot have been tampered with from inside a pod or lost to a file this
		// process could not write.
		if s.Protected != nil && s.Protected(entry.Name()) {
			continue
		}
		dir := filepath.Join(s.Root, InboxDir, entry.Name())

		// A live execution holds its inbox whether or not it has written to it lately,
		// and only the lease can say so. Checked before mtime, because an execution
		// that has cached nothing yet is exactly the one mtime judges most harshly.
		if s.LeaseTTL > 0 {
			// Lstat, not Stat: the step owns what .lease is, and a Stat would follow it
			// to whatever it names - another inbox's lease, or any file reachable from
			// here whose mtime happens to suit. A lease that is a link is not a lease.
			switch lease, err := os.Lstat(filepath.Join(dir, LeaseName)); {
			case err == nil && lease.Mode()&os.ModeSymlink != 0:
				failed = append(failed, fmt.Sprintf("%s: its lease is a symlink, which is not believed", entry.Name()))
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
				// Recorded and stepped over, never returned. The lease sits inside a
				// directory the step's own command can write, so a workflow can
				// replace it with a symlink pointing at itself and every Stat of it
				// answers ELOOP - and returning here would stop the sweep at that
				// inbox for good, on every pass, with everything after it never
				// reclaimed. A lease that cannot be read is simply not believed, so
				// this falls through to the mtime below and the inbox expires the
				// ordinary way.
				failed = append(failed, fmt.Sprintf("%s: reading its lease: %s", entry.Name(), err))
			}
		}

		// The inbox's own mtime moves whenever an entry is added to or removed from it,
		// so it tracks the execution's last write without having to walk inside. Read
		// once, above, where the kind was established.
		//
		// A future mtime is kept from pinning the inbox the same way, and for the same
		// reason: the step can touch its own inbox directory. Beyond the skew window it
		// is not a clock difference, so it is treated as expired rather than as
		// infinitely recent. A running execution is still safe - the lease above is
		// what holds a live inbox, and the agent writes that one.
		modified := info.ModTime()
		if modified.After(cutoff) && !modified.After(now().Add(maxClockSkew)) {
			// Retention bounds how long an entry outlives the pointer naming it, so an
			// inbox holding no entries has nothing to be held for. The agent makes one
			// for every execution the volume is enabled for - it cannot know whether a
			// parallel worker bundled later will cache - and most of those never write
			// anything, so holding them all for days would spend a great many inodes
			// on a volume the whole cluster shares.
			//
			// Still only once the lease has gone stale, which is what says the
			// execution is over. An inbox that is empty because its step has not
			// reached its cache yet is a running one, and the lease is what tells the
			// two apart.
			if s.LeaseTTL <= 0 || now().Sub(modified) < s.LeaseTTL {
				continue
			}
			empty, err := inboxIsEmpty(dir)
			if err != nil {
				failed = append(failed, fmt.Sprintf("%s: reading it: %s", entry.Name(), err))
				continue
			}
			if !empty {
				continue
			}
		}

		// Reported and stepped over rather than returned. Every inbox here is already
		// expired, so one the agent cannot remove - a directory left unwritable by the
		// step that made it, a file the volume will not release - must not stop the
		// sweep reaching the rest: that turns one stuck entry into a volume that never
		// reclaims anything again. The next pass tries it afresh.
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			failed = append(failed, fmt.Sprintf("%s: %s", entry.Name(), err))
		}
	}

	// Returned together, after everything that could be swept has been. Continuing is
	// what keeps one stuck inbox from stranding the rest; reporting is what keeps it
	// from being stuck in silence, with the volume filling and nothing said.
	if len(failed) > 0 {
		return fmt.Errorf("could not remove %d expired cache inbox(es): %s",
			len(failed), strings.Join(failed, "; "))
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

	// Everything below goes through a root opened on the inbox, never through a path
	// the agent assembles itself.
	//
	// The inbox is writable by the step, so the step owns what .lease is. Left to
	// os.Chtimes and os.OpenFile, both of which follow the final symlink, a workflow
	// could point .lease at an absolute path and have the **agent** stamp - or create -
	// a file inside its own container, which the step has no access to and far fewer
	// privileges than. os.Root refuses a link that leaves the root, so the worst a link
	// can name is something else in this same inbox; the Lstat below refuses even that,
	// since a lease is a file this process writes and nothing else should be.
	//
	// Opening the root fails when the inbox is not there, which is a swept or
	// never-made inbox - the error says so, rather than building one nothing reads.
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()

	switch info, lstatErr := root.Lstat(LeaseName); {
	case lstatErr == nil && info.Mode()&os.ModeSymlink != 0:
		// Not followed and not trusted: replaced. Failing instead would leave the
		// inbox pinned for as long as its mtime held out, which is the outcome the
		// step would be angling for.
		if rmErr := root.Remove(LeaseName); rmErr != nil {
			return rmErr
		}
	case lstatErr != nil && !os.IsNotExist(lstatErr):
		return lstatErr
	}

	// Renewed by writing to it, not by setting its timestamp.
	//
	// Each deployment holds its own leader election, so an api and a runner sharing one
	// volume both renew, as different users - and whichever did not create this lease
	// is not its owner. Setting an explicit timestamp is the owner's privilege, where a
	// write asks only for the write bit the mode below grants, and updates the mtime
	// just the same. Chtimes would have left the lease renewable only by the process
	// that made it, so if that one stopped the lease went stale under a live execution
	// and the sweep took its inbox.
	//
	// O_CREATE and not MkdirAll: a missing parent is a swept or never-made inbox, and
	// the error says so rather than building one nothing reads.
	f, err := root.OpenFile(LeaseName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, SharedFileMode)
	if os.IsPermission(err) {
		// Replaced rather than given up on, for the same reason as the symlink above.
		// The step holds this inbox's mount and can make .lease itself, as whatever
		// user the workflow runs and under its own umask: one left 0600 by uid 1000 is
		// one an agent running as another user can never write. Giving up would fail
		// the renewal for as long as the execution ran, and a lease going stale under a
		// live execution is precisely what has its inbox swept from under it - after
		// which its pod writes through a subPath to an inode nothing can reach and
		// publishes a pointer naming a path that is gone. Unlinking needs the write bit
		// on the inbox, not ownership of the file, and the inbox is 0777.
		if rmErr := root.Remove(LeaseName); rmErr != nil && !os.IsNotExist(rmErr) {
			return err
		}
		f, err = root.OpenFile(LeaseName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, SharedFileMode)
	}
	if err != nil {
		return err
	}
	_, writeErr := f.Write([]byte("lease\n"))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}

	// OpenFile's mode is filtered by the umask, so it is set explicitly: under 0077 a
	// lease would arrive 0600 and no other agent could renew it. Only the owner may
	// chmod, which here means only whoever created it - so a refusal is one this
	// process cannot act on and is not an error, the creator having already decided.
	if err := root.Chmod(LeaseName, SharedFileMode); err != nil && !os.IsPermission(err) {
		return err
	}
	return nil
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

// inboxIsEmpty reports whether an inbox holds no entries, the lease not counting as one.
//
// The lease is written by the agent rather than by the execution, so an inbox holding
// only that has had nothing cached into it and names nothing any pointer could follow.
func inboxIsEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Swept by another agent between the listing and here, which is as empty
			// as it gets.
			return true, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.Name() != LeaseName {
			return false, nil
		}
	}
	return true, nil
}
