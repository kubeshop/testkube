package volume

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// CopyLimits bound what a copy will move, so a malformed or hostile entry cannot fill
// the destination or walk forever.
//
// They mirror the guards the archive form carried, because the entry is still written
// by whoever populated the cache - under an environment-scoped cache, another workflow.
type CopyLimits struct {
	MaxTotalBytes int64
	MaxEntries    int
}

var (
	// ErrTooLarge reports that a copy stopped at MaxTotalBytes.
	ErrTooLarge = errors.New("cache entry is larger than the limit")
	// ErrTooManyEntries reports that a copy stopped at MaxEntries.
	ErrTooManyEntries = errors.New("cache entry holds more files than the limit")
)

// remaining is how many bytes a copy may still write.
//
// A MaxTotalBytes of zero means unbounded, matching MaxEntries and matching what a
// zero-valued CopyLimits reads as. Subtracting from zero instead would make "no limit"
// the tightest limit there is, and refuse the first byte of every copy.
func remaining(max, used int64) int64 {
	if max <= 0 {
		return math.MaxInt64
	}
	return max - used
}

// RestoreTree copies an entry's mirrored tree onto the filesystem.
//
// src is <entry>/root, where every path is an absolute container path with its leading
// separator dropped. Each declared path is restored separately and **through an os.Root
// opened on that path**, which is what confines the write: a symlink already sitting
// inside a declared path cannot redirect a write outside it, because os.Root refuses to
// traverse one that leaves the root. Writing by absolute name instead would follow it,
// and an entry may have been written by another workflow under an environment-scoped
// key - so that link is attacker-controlled input, not a local detail.
//
// Restoring per declared path also does the job the archive form needed an allowlist
// for: anything in the entry outside the declared paths is simply never reached.
//
// It reports whether it wrote anything before failing. A caller that has written
// something has a half-restored tree and must clear the declared paths; one that has
// not can leave them alone, which matters because a declared path may hold a checkout
// the step still needs.
func RestoreTree(src *os.Root, declaredPaths []string, limits CopyLimits) (wrote bool, err error) {
	var (
		total    int64
		entries  int
		wanted   int
		restored int
	)

	for _, declared := range coverPaths(declaredPaths) {
		dest := path.Clean(declared)
		if dest == "" || dest == "/" || dest == "." {
			continue
		}
		wanted++

		// Where this path lives inside the entry: the same absolute name with the
		// leading separator dropped, which is how the entry mirrors the filesystem.
		within := strings.TrimPrefix(dest, "/")
		if _, statErr := src.Stat(within); statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				// The entry does not carry this path. A smaller entry than the step
				// declared is a normal thing to restore, not a fault.
				continue
			}
			// Anything else - a permission problem, an I/O error, a volume that went
			// away mid-restore - is not evidence that the path is absent, and reading
			// it as such would report a hit for a path nothing was restored to. The
			// entry is exact, so the save stage would then skip replacing it, and a
			// declared path that did restore leaves a tree that looks whole and is
			// not. Report it and let the caller clear up and call it a miss.
			return wrote, statErr
		}
		restored++

		dst, openErr := openDeclaredRoot(dest)
		if openErr != nil {
			return wrote, openErr
		}

		didWrite, copyErr := restoreInto(src, within, dst, &total, &entries, limits)
		dst.Close()
		if didWrite {
			wrote = true
		}
		if copyErr != nil {
			return wrote, copyErr
		}
	}

	// An entry that carries none of the paths this step asked for is not a hit, even
	// though every lookup said it was. A key does not describe the paths it was saved
	// with - it is whatever the workflow templated, commonly a lockfile hash - so an
	// environment-scoped key shared by workflows that cache different directories, or
	// a workflow that changes its paths without changing its key, lands here.
	//
	// Returning success would report an exact hit that restored nothing, and an exact
	// hit tells the save stage there is nothing to replace, so the step would reinstall
	// on every execution with the entry still claiming to hold what it does not. The
	// archive backend refuses the same mismatch through UnpackTarball's allowed roots.
	//
	// Some but not all is fine, and stays a hit: SaveTree skips a declared path that
	// did not exist when the entry was written, so an entry smaller than the step
	// declared is the ordinary shape of one.
	if wanted > 0 && restored == 0 {
		return wrote, ErrEntryHoldsNoDeclaredPath
	}
	return wrote, nil
}

// ErrEntryHoldsNoDeclaredPath reports an entry that carries none of the paths the step
// asked to restore, which the caller reports as a miss rather than an empty hit.
var ErrEntryHoldsNoDeclaredPath = errors.New("the entry holds none of the declared cache paths")

// ErrDeclaredPathIsSymlink reports a declared cache path that is - or is reached
// through - a symlink.
var ErrDeclaredPathIsSymlink = errors.New("declared cache path is reached through a symlink")

// coverPaths drops declared paths that another declared path already contains.
//
// A workflow may legitimately declare both /data/deps and /data/deps/packages. Walking
// each in turn would then visit the nested tree twice: storing it twice on the volume,
// and counting it twice against the size and entry limits. The archive form does not,
// because its walker crosses the filesystem once and matches every file against all the
// patterns - so without this the two backends disagree about how big the same cache is,
// and the volume can refuse a cache the archive would have taken.
//
// Sorting puts a parent immediately before everything beneath it, so comparing each
// path against the last one kept is enough to drop the whole nested run.
func coverPaths(paths []string) []string {
	cleaned := make([]string, 0, len(paths))
	for _, p := range paths {
		c := path.Clean(p)
		if c == "" || c == "/" || c == "." {
			continue
		}
		cleaned = append(cleaned, c)
	}
	sort.Strings(cleaned)

	covered := make([]string, 0, len(cleaned))
	for _, c := range cleaned {
		if n := len(covered); n > 0 {
			last := covered[n-1]
			// The trailing separator is what keeps /data/deps2 from looking like it
			// sits under /data/deps.
			if c == last || strings.HasPrefix(c, last+"/") {
				continue
			}
		}
		covered = append(covered, c)
	}
	return covered
}

// openDeclaredRoot opens a declared path as a root, having first established that it is
// a real directory reached only through real directories.
//
// os.Root confines what happens *after* it is opened; opening it does not confine
// itself. os.OpenRoot resolves the name it is given, final symlink included, so a
// declared path that is a link to somewhere else yields a root at that somewhere else
// and every subsequent write - confined, correctly, to the wrong place - lands outside
// the declared path. Worse, the cleanup that a failed restore performs would then empty
// the link rather than what was written through it.
//
// So each component is checked before it is traversed, and missing ones are created as
// real directories. A step that has made one of them a symlink gets a miss, which is
// the same answer any other unusable declared path gets.
func openDeclaredRoot(dest string) (*os.Root, error) {
	parts := strings.Split(strings.Trim(dest, "/"), "/")

	// Where the walk starts has to be where os.OpenRoot below will look, or this
	// checks and creates one directory and then opens another.
	//
	// A declared path is relative whenever the step's own image decides the working
	// directory, which is the common case: mountCachePaths resolves a relative path
	// against the working directory only where the bundle already knows it, and leaves
	// it alone otherwise. Walking from the root then created /node_modules, left
	// ./node_modules uncreated, and OpenRoot failed on it - so the volume never
	// restored a relative path at all. It also ran the symlink guard below against a
	// path that was not the one about to be written to, which is the whole point of
	// the guard.
	//
	// Relative to the process, deliberately: SaveTree walks the same spelling from the
	// same place, so the two agree about what an entry holds.
	current := "."
	if strings.HasPrefix(dest, "/") {
		current = "/"
	}

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = path.Join(current, part)
		native := filepath.FromSlash(current)

		info, err := os.Lstat(native)
		switch {
		case os.IsNotExist(err):
			if mkErr := os.Mkdir(native, 0o777); mkErr != nil && !os.IsExist(mkErr) {
				return nil, mkErr
			}
		case err != nil:
			return nil, err
		case info.Mode()&os.ModeSymlink != 0:
			return nil, fmt.Errorf("%s: %w", current, ErrDeclaredPathIsSymlink)
		case !info.IsDir():
			return nil, fmt.Errorf("%s is not a directory", current)
		}
	}
	return os.OpenRoot(filepath.FromSlash(dest))
}

// restoreInto copies one declared path's subtree out of the entry and into dst.
func restoreInto(src *os.Root, within string, dst *os.Root, total *int64, entries *int, limits CopyLimits) (bool, error) {
	var wrote bool

	err := fs.WalkDir(src.FS(), within, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Relative to the declared path, which is what dst is rooted at.
		rel := strings.TrimPrefix(strings.TrimPrefix(name, within), "/")
		if rel == "" {
			return nil
		}

		// Directories are counted, and before they are made, exactly as the save counts
		// them - a tree that passed the limit going in has to pass it coming out, or
		// the entry would store and then be refused by every restore of it, under a
		// key no later run could replace. Nothing else bounds a directory: MaxTotalBytes
		// weighs file contents and a directory has none, so an entry of millions of
		// empty directories would otherwise cost an inode and a mkdir apiece on every
		// restore.
		//
		// Made still counts as written: wrote is what tells the caller to clear the
		// declared paths after a failure, so it has to mean "this touched the
		// filesystem", not "this copied bytes". A restore that creates a directory tree
		// and then stops at the limit would otherwise report nothing written and leave
		// that tree behind for the install to find.
		if d.IsDir() {
			*entries++
			if limits.MaxEntries > 0 && *entries > limits.MaxEntries {
				return ErrTooManyEntries
			}
			wrote = true
			return dst.MkdirAll(rel, 0o777)
		}

		*entries++
		if limits.MaxEntries > 0 && *entries > limits.MaxEntries {
			return ErrTooManyEntries
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// A symlink is recreated as a symlink rather than followed. Creating one is
		// safe whatever it points at - it is only a name - and the os.Root this writes
		// through is what stops anything later following it out of the declared path.
		// Dropping them instead would restore an incomplete tree that still answers as
		// an exact hit, so the step would never repair it: node_modules/.bin is
		// entirely symlinks.
		if info.Mode()&fs.ModeSymlink != 0 {
			target, readErr := src.Readlink(name)
			if readErr != nil {
				return readErr
			}
			// Set before the first mutation rather than after the last one: the Remove
			// below can succeed and the Symlink then fail, which has changed the tree
			// even though nothing was created.
			wrote = true
			if mkErr := dst.MkdirAll(path.Dir(rel), 0o777); mkErr != nil && path.Dir(rel) != "." {
				return mkErr
			}
			// An entry restored over an existing tree may find the link already there.
			_ = dst.Remove(rel)
			return dst.Symlink(target, rel)
		}

		if !info.Mode().IsRegular() {
			// Sockets, devices and pipes cannot be reproduced meaningfully and no
			// dependency tree needs them.
			return nil
		}

		// Likewise set before the copy, not from the byte count it returns: an empty
		// file is still a file the install would find, and copyIntoRoot removes what
		// is already at the path before creating anything, so even a copy that fails
		// immediately may have changed the tree.
		wrote = true
		n, copyErr := copyIntoRoot(src, name, dst, rel, info, remaining(limits.MaxTotalBytes, *total))
		*total += n
		return copyErr
	})
	return wrote, err
}

// SaveTree copies the declared paths into a staged entry, mirroring the filesystem.
//
// dst is the staging directory's <entry>/root. Paths that do not exist are skipped: a
// step may declare a cache path it never creates, and that is a smaller entry rather
// than a failure.
//
// It returns the total bytes written - which is what the pointer carries and what a
// quota would be refused against - and how many files it found. The count is separate
// because a tree of empty files is still a tree, where zero bytes alone would read as
// nothing to save and publish an entry that answers every later run with a hit that
// restores nothing.
func SaveTree(dst string, paths []string, limits CopyLimits) (int64, int, error) {
	var (
		total int64
		// counted bounds the work this entry will cost - every inode, directories
		// included - and is what MaxEntries applies to.
		//
		// content is what the caller means by an empty entry, and counts only files and
		// links. A tree of nothing but directories restores nothing, so publishing it
		// under an immutable key would answer every later run with a hit holding no
		// files at all.
		counted int
		content int
	)
	for _, p := range coverPaths(paths) {
		src := path.Clean(p)
		if src == "" || src == "/" || src == "." {
			continue
		}
		base := filepath.Join(dst, filepath.FromSlash(strings.TrimPrefix(src, "/")))

		// filepath.Walk lstats, so a symlink arrives as a symlink rather than as
		// whatever it points at - which is what lets the entry carry the link itself.
		err := filepath.Walk(filepath.FromSlash(src), func(name string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			rel, relErr := filepath.Rel(filepath.FromSlash(src), name)
			if relErr != nil {
				return relErr
			}
			target := base
			if rel != "." {
				target = filepath.Join(base, rel)
			}

			switch {
			case info.IsDir():
				// Counted, and before it is made. A directory costs an inode on the
				// shared volume and a mkdir on every restore, and nothing else here
				// bounds one: MaxTotalBytes weighs file contents, of which a directory
				// has none. A step controls what is under its own cached paths, so a
				// tree of millions of empty directories would otherwise be copied onto
				// a volume every execution shares, and recreated by every restore.
				//
				// The root of a declared path is not skipped here the way it is on the
				// restore side, because rel == "." is that root and it is one
				// directory either way.
				counted++
				if limits.MaxEntries > 0 && counted > limits.MaxEntries {
					return ErrTooManyEntries
				}
				return os.MkdirAll(target, 0o777)

			case info.Mode()&os.ModeSymlink != 0:
				counted++
				content++
				if limits.MaxEntries > 0 && counted > limits.MaxEntries {
					return ErrTooManyEntries
				}
				link, readErr := os.Readlink(name)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						return nil
					}
					return readErr
				}
				if mkErr := os.MkdirAll(filepath.Dir(target), 0o777); mkErr != nil {
					return mkErr
				}
				// The link is stored as written, relative or absolute. Resolving it
				// here would turn a relative link - which is what a dependency tree
				// uses, and what survives being restored somewhere else - into one
				// naming this pod's filesystem.
				return os.Symlink(link, target)

			case info.Mode().IsRegular():
				counted++
				content++
				if limits.MaxEntries > 0 && counted > limits.MaxEntries {
					return ErrTooManyEntries
				}
				n, copyErr := copyOut(name, target, info, remaining(limits.MaxTotalBytes, total))
				total += n
				return copyErr

			default:
				// See RestoreTree: sockets, devices and pipes carry nothing a restore
				// could reproduce.
				return nil
			}
		})
		if err != nil {
			return total, content, err
		}
	}
	return total, content, nil
}

// copyIntoRoot writes one regular file out of the entry, through the destination root.
func copyIntoRoot(src *os.Root, name string, dst *os.Root, rel string, info fs.FileInfo, budget int64) (int64, error) {
	if budget <= 0 {
		return 0, ErrTooLarge
	}
	if dir := path.Dir(rel); dir != "." {
		if err := dst.MkdirAll(dir, 0o777); err != nil {
			return 0, err
		}
	}

	in, err := src.Open(name)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	// O_NOFOLLOW on top of the root: the root already refuses a link out of it, and
	// this refuses one that stays inside, so a restore cannot be made to write through
	// any pre-existing link at all.
	//
	// 0666 before umask, for the reason the state file uses it: the stages, and whole
	// executions, may run as different users, and a restored tree a later step cannot
	// read is worse than a miss.
	_ = dst.Remove(rel)
	out, err := dst.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, io.LimitReader(in, budget))
	closeErr := out.Close()
	if err != nil {
		return n, err
	}
	if closeErr != nil {
		return n, closeErr
	}
	if n == budget && info.Size() > n {
		return n, ErrTooLarge
	}
	// Mode is carried so an executable in a dependency tree stays executable.
	return n, dst.Chmod(rel, info.Mode().Perm()|0o666)
}

// copyOut writes one regular file from the filesystem into the staged entry.
func copyOut(name, target string, info os.FileInfo, budget int64) (int64, error) {
	if budget <= 0 {
		return 0, ErrTooLarge
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o777); err != nil {
		return 0, err
	}

	in, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			// The tree changed under the walk, which a build directory does. Skipping
			// is right: the entry is a snapshot, not a transaction.
			return 0, nil
		}
		return 0, err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, io.LimitReader(in, budget))
	closeErr := out.Close()
	if err != nil {
		return n, err
	}
	if closeErr != nil {
		return n, closeErr
	}
	if n == budget && info.Size() > n {
		return n, ErrTooLarge
	}
	return n, os.Chmod(target, info.Mode().Perm()|0o666)
}
