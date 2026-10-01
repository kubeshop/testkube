package volume

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

// RestoreTree copies an entry's mirrored tree onto the filesystem.
//
// src is <entry>/root, where every path is an absolute container path with its leading
// separator dropped. allowedRoots is the set of paths the step declared, and anything
// outside them is skipped rather than written - the same rule the archive form applied,
// and for the same reason: an entry may have been written by another workflow, so a
// file in the repository checkout or on a shared internal volume must not be restored
// somewhere the step never declared and the cleanup would not reach.
//
// It reports whether it wrote anything before failing. A caller that has written
// something has a half-restored tree and must clear the declared paths; one that has
// not can leave them alone, which matters because a declared path may hold a checkout
// the step still needs.
func RestoreTree(src *os.Root, allowedRoots []string, limits CopyLimits) (wrote bool, err error) {
	allowed := cleanRoots(allowedRoots)

	var (
		total   int64
		entries int
	)
	err = fs.WalkDir(src.FS(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}

		target := "/" + name
		if !permits(target, allowed) {
			// Skipping a whole directory is what keeps an entry naming a tree outside
			// the declared paths from costing a walk of all of it.
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		entries++
		if limits.MaxEntries > 0 && entries > limits.MaxEntries {
			return ErrTooManyEntries
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o777)
		case info.Mode().IsRegular():
			n, err := copyFile(src, name, target, info, limits.MaxTotalBytes-total)
			if n > 0 {
				wrote = true
			}
			total += n
			return err
		default:
			// Sockets, devices and symlinks are skipped rather than refused: a
			// dependency tree legitimately carries symlinks (npm bins), and following
			// one when copying back out is how an entry escapes its declared paths.
			// Dropping them keeps the entry to plain files, which is all a restore can
			// reproduce faithfully anyway.
			return nil
		}
	})
	if err != nil {
		return wrote, err
	}
	return wrote, nil
}

// SaveTree copies the declared paths into a staged entry, mirroring the filesystem.
//
// dst is the staging directory's <entry>/root. Paths that do not exist are skipped: a
// step may declare a cache path it never creates, and that is a smaller entry rather
// than a failure.
//
// It returns the total bytes written, which is what the pointer carries and what a
// quota would be refused against.
func SaveTree(dst string, paths []string, limits CopyLimits) (int64, error) {
	var (
		total   int64
		entries int
	)
	for _, p := range paths {
		src := path.Clean(p)
		if src == "" || src == "/" {
			continue
		}
		base := filepath.Join(dst, filepath.FromSlash(strings.TrimPrefix(src, "/")))

		err := filepath.Walk(filepath.FromSlash(src), func(name string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			rel, err := filepath.Rel(filepath.FromSlash(src), name)
			if err != nil {
				return err
			}
			target := base
			if rel != "." {
				target = filepath.Join(base, rel)
			}

			switch {
			case info.IsDir():
				return os.MkdirAll(target, 0o777)
			case info.Mode().IsRegular():
				entries++
				if limits.MaxEntries > 0 && entries > limits.MaxEntries {
					return ErrTooManyEntries
				}
				n, err := copyOut(name, target, info, limits.MaxTotalBytes-total)
				total += n
				return err
			default:
				// See RestoreTree: symlinks and specials are not carried, so an entry
				// holds only what a restore can faithfully reproduce.
				return nil
			}
		})
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// copyFile writes one regular file out of the entry onto the filesystem.
func copyFile(src *os.Root, name, target string, info fs.FileInfo, budget int64) (int64, error) {
	if budget <= 0 {
		return 0, ErrTooLarge
	}
	if err := os.MkdirAll(path.Dir(target), 0o777); err != nil {
		return 0, err
	}

	in, err := src.Open(name)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	// 0666 before umask, for the reason the state file uses it: the stages, and whole
	// executions, may run as different users, and a restored tree a later step cannot
	// read is worse than a miss.
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
	// Mode is carried so an executable in a dependency tree stays executable.
	if err := os.Chmod(target, info.Mode().Perm()|0o666); err != nil {
		return n, err
	}
	return n, nil
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
		return n, fmt.Errorf("%w (at %s)", ErrTooLarge, name)
	}
	return n, os.Chmod(target, info.Mode().Perm()|0o666)
}

func cleanRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		if r == "" {
			continue
		}
		out = append(out, path.Clean(r))
	}
	return out
}

// permits reports whether a restored path falls inside one of the declared paths.
func permits(target string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, root := range allowed {
		if target == root || strings.HasPrefix(target, root+"/") {
			return true
		}
		// A parent of a declared path has to be walked to reach it.
		if strings.HasPrefix(root, target+"/") {
			return true
		}
	}
	return false
}
