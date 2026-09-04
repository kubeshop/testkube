package common

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
)

// copyBufferSize is the unit an entry is read and written in.
//
// The default io.Copy buffer is 32 KiB and is allocated per call. A dependency cache is
// hundreds of thousands of entries, so that is both a syscall per 32 KiB and a 32 KiB
// allocation per file; one reused buffer removes the second entirely and divides the
// first by 32.
const copyBufferSize = 1 << 20

var (
	relativeCheckRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)
)

// WriteTarball packs files matching the patterns, resolved against dirPath.
func WriteTarball(stream io.Writer, dirPath string, files []string) error {
	_, err := WriteTarballFrom(stream, dirPath, files, nil)
	return err
}

// WriteTarballFrom packs files matching the patterns, rooted at dirPath and restricted
// to the given mounts. Passing no mounts restricts them to dirPath itself.
//
// The mounts argument exists for the dependency cache, whose paths legitimately live in
// several volumes at once - node_modules under the repository and, say, /root/.m2 on its
// own emptyDir. Those have no common ancestor but the filesystem root, so packing them
// needs a root of "/" together with the set of directories that may actually be read;
// the walker would otherwise reduce the root to "/" and offer no way to bound it.
// Entry names stay relative to the root, so UnpackTarball's own path checks still apply.
//
// Returns how many entries were written, because "matched nothing" and "matched
// something" are not the same outcome to the caller and the archive itself cannot be
// told apart by size: an empty tarball is a valid one, a few dozen bytes long.
func WriteTarballFrom(stream io.Writer, dirPath string, files []string, mounts []string) (entries int, err error) {
	// Ensure the absolute path
	if !filepath.IsAbs(dirPath) {
		dirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return 0, errors.Wrap(err, "failed to build absolute path for writing tarball")
		}
	}
	if len(mounts) == 0 {
		mounts = []string{dirPath}
	}

	// Prepare files archive.
	//
	// klauspost/compress is a drop-in for compress/gzip and its encoder is around
	// seventeen times faster here at the same ratio, measured on a corpus mixed to match
	// a Go module cache - which is half already-compressed module zips, so most of what
	// the standard library's default level spends its time on cannot pay off. Packing
	// 2.2 GiB went from tens of seconds to a few.
	gzipStream := gzip.NewWriter(stream)
	tarStream := tar.NewWriter(gzipStream)
	defer gzipStream.Close()
	defer tarStream.Close()

	// Append all the files
	walker, err := artifacts.CreateWalker(files, mounts, dirPath)
	if err != nil {
		return 0, err
	}
	buffer := make([]byte, copyBufferSize)
	err = walker.Walk(os.DirFS("/"), func(path string, file fs.File, stat fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
			return nil
		}
		// The walker opens every entry and hands it over without closing it. For a
		// dependency cache that is a few hundred thousand descriptors held until the
		// garbage collector runs each file's finalizer - which is what closes an
		// *os.File nobody closed - so the cost lands as GC pressure during the pack and
		// the run depends on the container's descriptor limit being generous.
		if file != nil {
			defer file.Close()
		}

		// Append the file to the archive
		name := stat.Name()
		link := name
		isSymlink := stat.Mode()&fs.ModeSymlink != 0
		if isSymlink {
			link, err = os.Readlink(filepath.Join(dirPath, path))
			if err != nil {
				fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading link: %s\n", path, err.Error())
				return nil
			}
		}

		// Build the data
		header, err := tar.FileInfoHeader(stat, link)
		if err != nil {
			return err
		}
		header.Name = path
		err = tarStream.WriteHeader(header)
		if err != nil {
			return err
		}

		// Copy the contents for regular files, through one reused buffer. io.Copy
		// allocates a 32 KiB one per call, which across a dependency cache's file count
		// is gigabytes of garbage for no benefit. readOnly hides the file's WriteTo,
		// which io.CopyBuffer prefers over the buffer it was handed, and which falls
		// back to allocating its own for a destination like a tar writer.
		if !isSymlink {
			if _, err = io.CopyBuffer(tarStream, readOnly{file}, buffer); err != nil {
				return err
			}
		}

		entries++
		return nil
	})
	return entries, err
}

// UnpackOptions bounds what an archive may expand into.
//
// The zero value applies no limit, which is what the pod-to-pod transfer path wants:
// there the archive was produced by a sibling container in the same execution, so its
// size is already the caller's own doing. An archive from outside that trust domain -
// a dependency cache written by an earlier, possibly different workflow - should be
// bounded, because nothing else stops it filling the pod's volume.
type UnpackOptions struct {
	// MaxTotalBytes caps the decompressed total across every entry. 0 means unlimited.
	MaxTotalBytes int64
	// MaxEntries caps how many entries are extracted. 0 means unlimited.
	MaxEntries int
	// AllowedRoots restricts extraction to entries landing inside one of these
	// container paths. Empty means the whole destination is writable.
	AllowedRoots []string
}

type UnpackOption func(*UnpackOptions)

func WithMaxTotalBytes(n int64) UnpackOption {
	return func(o *UnpackOptions) { o.MaxTotalBytes = n }
}

func WithMaxEntries(n int) UnpackOption {
	return func(o *UnpackOptions) { o.MaxEntries = n }
}

// WithAllowedRoots restricts extraction to the given container paths.
//
// Confining to the destination directory is not enough on its own when that directory
// is the filesystem root, which is what unpacking a dependency cache needs: the entries
// name paths spread across several volumes, so there is no single subdirectory to
// extract into. os.Root then bounds the extraction to "/", which bounds nothing.
//
// An archive is written by whoever populated the cache, and under an environment-scoped
// cache that is another workflow. So the reader states which paths it asked to have
// restored, and anything else in the archive - a file in the repository checkout, a
// binary on the shared internal volume - is refused rather than written somewhere the
// step never declared and the cleanup would not reach.
func WithAllowedRoots(roots ...string) UnpackOption {
	return func(o *UnpackOptions) {
		for _, root := range roots {
			if root == "" {
				continue
			}
			o.AllowedRoots = append(o.AllowedRoots, path.Clean(root))
		}
	}
}

// permits reports whether an entry resolving to containerPath may be written.
func (o UnpackOptions) permits(containerPath string) bool {
	if len(o.AllowedRoots) == 0 {
		return true
	}
	for _, root := range o.AllowedRoots {
		if containerPath == root || strings.HasPrefix(containerPath, root+"/") {
			return true
		}
	}
	return false
}

// normalizeEntryName turns a tar entry's name into the one relative form that the rest
// of the extraction uses, or reports it as unsafe.
//
// A tar entry name is slash-separated by the format, whatever host wrote it, so this
// judges it with `path` and never with `filepath`. Deliberately no ToSlash: that is a
// host-dependent rewrite, and applying it made the decision depend on the machine doing
// the unpacking - "\abs" survives untouched on Linux, where it is an ordinary one-segment
// filename that happens to contain a backslash, and becomes "/abs" on Windows, where the
// check then read it as absolute. Neither outcome is wrong about its own platform, which
// is exactly the problem: the same archive must be judged the same way everywhere.
//
// So a backslash is treated as what it is in a POSIX filename - a character - and the
// same string that was validated is the one every os.Root call and the allowlist get.
func normalizeEntryName(raw string) (string, error) {
	unsafe := func() error {
		return fmt.Errorf("unsafe file path in the tarball: %s", raw)
	}

	if raw == "" {
		return "", unsafe()
	}

	// Absolute by the only rule a tar name has.
	if strings.HasPrefix(raw, "/") {
		return "", unsafe()
	}

	// A volume or drive-qualified name is not a relative path anywhere, and path.Join
	// would silently treat it as one.
	if first, _, _ := strings.Cut(raw, "/"); strings.Contains(first, ":") {
		return "", unsafe()
	}

	// Any parent traversal at all, which is stricter than the extraction strictly needs
	// - "a/../b" resolves inside the root - but keeps a name that reads as an escape
	// from being accepted on the grounds that it happens to cancel out.
	if relativeCheckRe.MatchString(raw) {
		return "", unsafe()
	}

	cleaned := path.Clean(raw)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", unsafe()
	}
	return cleaned, nil
}

// UnpackTarball extracts an archive into dirPath.
//
// Extraction goes through os.Root, so every path is resolved inside dirPath by the
// kernel. That is what makes an archive from an untrusted producer safe to unpack: the
// header.Name checks below catch a name that is itself an escape, but they cannot catch
// an escape assembled from two innocent-looking entries - a symlink 'dep -> /' followed
// by a regular file 'dep/etc/x', whose own name contains neither '..' nor a leading
// slash. Writing that second entry with os.OpenFile would follow the link straight out
// of the destination. os.Root refuses it instead, and refuses the whole class with it.
func UnpackTarball(dirPath string, stream io.Reader, opts ...UnpackOption) error {
	var options UnpackOptions
	for _, opt := range opts {
		opt(&options)
	}

	// os.OpenRoot needs the directory to exist, whereas the previous implementation
	// created it implicitly on the first entry.
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return errors.Wrapf(err, "%s: create destination directory", dirPath)
	}
	root, err := os.OpenRoot(dirPath)
	if err != nil {
		return errors.Wrapf(err, "%s: open destination directory", dirPath)
	}
	defer root.Close()

	// Process the files
	uncompressedStream, err := gzip.NewReader(stream)
	if err != nil {
		return errors.Wrap(err, "start reading gzip")
	}
	tarReader := tar.NewReader(uncompressedStream)

	var entries int
	var written int64

	// Where the entries land, as the allowlist expresses it. Hoisted because it does not
	// depend on the entry, and this loop runs once per file.
	destination := path.Clean(filepath.ToSlash(dirPath))

	// Directories already created. Our own archives carry no directory entries at all -
	// the walker skips them - so every parent has to be created from the file paths, and
	// without this that is one MkdirAll per file rather than one per directory. It is the
	// dominant cost of restoring a dependency cache: a Go module cache is a few hundred
	// thousand files across a few tens of thousands of directories, and os.Root resolves
	// every component of every call itself in order to keep the extraction confined.
	created := make(map[string]struct{})
	ensureDir := func(name string) error {
		if name == "" || name == "." || name == "/" {
			return nil
		}
		if _, ok := created[name]; ok {
			return nil
		}
		if err := root.MkdirAll(name, 0755); err != nil {
			return err
		}
		// Every ancestor exists now too, so none of them needs a call of its own later.
		for dir := name; dir != "" && dir != "." && dir != "/"; dir = path.Dir(dir) {
			created[dir] = struct{}{}
		}
		return nil
	}

	buffer := make([]byte, copyBufferSize)

	// Unpack them
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "get next entry from tarball")
		}
		// Normalize once, then validate what was normalized. Checking the raw name with
		// host rules and then handing the slash-separated form to os.Root would make the
		// guarantee depend on the platform: "\abs" is not absolute to filepath.IsAbs on
		// Linux, and on Windows it is not absolute either, yet ToSlash turns it into
		// "/abs". Everything below - the allowlist and every os.Root call - now sees the
		// same string the check saw.
		name, err := normalizeEntryName(header.Name)
		if err != nil {
			return err
		}

		entries++
		if options.MaxEntries > 0 && entries > options.MaxEntries {
			return fmt.Errorf("tarball holds more than %d entries", options.MaxEntries)
		}

		// Kept for error messages and for the mode; os.Root does the resolving.
		filePath := filepath.Join(dirPath, filepath.FromSlash(name))

		// Where this entry would land inside the container, which is what the caller's
		// allowlist is expressed in.
		if !options.permits(path.Join(destination, name)) {
			return fmt.Errorf("tarball entry %s is outside the paths this step asked to restore", name)
		}

		// Only the permission bits are honoured. header.Mode is attacker-controlled in
		// an archive from another execution, and nothing in a dependency tree needs
		// setuid, setgid or sticky.
		mode := os.FileMode(header.Mode).Perm() & 0o777

		switch header.Typeflag {
		case tar.TypeDir:
			err := ensureDir(name)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory", filePath)
			}
		case tar.TypeReg:
			err := ensureDir(path.Dir(name))
			if err != nil {
				return errors.Wrapf(err, "%s: create directory tree", filePath)
			}
			outFile, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
			if err != nil {
				return errors.Wrapf(err, "%s: create file", filePath)
			}
			// header.Size is advisory - a tar header may understate what follows - so
			// the limit is enforced on what is actually read, not on what is declared.
			// writeOnly hides the file's ReadFrom, which io.CopyBuffer would otherwise
			// prefer over the buffer it was handed - and which, for a source that is not
			// a file, ends in a generic copy through a freshly allocated 32 KiB one.
			destinationFile := writeOnly{outFile}
			var copied int64
			if options.MaxTotalBytes > 0 {
				remaining := options.MaxTotalBytes - written
				copied, err = io.CopyBuffer(destinationFile, io.LimitReader(tarReader, remaining+1), buffer)
				if err == nil && copied > remaining {
					_ = outFile.Close()
					return fmt.Errorf("tarball expands to more than %d bytes", options.MaxTotalBytes)
				}
			} else {
				copied, err = io.CopyBuffer(destinationFile, tarReader, buffer)
			}
			if err != nil {
				_ = outFile.Close()
				return errors.Wrapf(err, "%s: write file", filePath)
			}
			written += copied
			_ = outFile.Close()
		case tar.TypeSymlink:
			err := ensureDir(path.Dir(name))
			if err != nil {
				return errors.Wrapf(err, "%s: create directory tree", filePath)
			}
			if _, statErr := root.Lstat(name); statErr == nil {
				if rmErr := root.Remove(name); rmErr != nil {
					return errors.Wrapf(rmErr, "%s: replace existing entry before symlink", filePath)
				}
			}
			// The link may point anywhere, including outside the root - creating it is
			// not itself an escape, and pnpm and yarn workspaces rely on symlinks. What
			// matters is that any later write *through* it is refused above.
			err = root.Symlink(header.Linkname, name)
			if err != nil {
				return errors.Wrapf(err, "%s: create symlink", filePath)
			}
		default:
			return fmt.Errorf("unknown entry type in the transferred archive: '%x' in %s", header.Typeflag, filePath)
		}
	}
	return nil
}

// readOnly exposes only Read, for the same reason writeOnly exposes only Write.
type readOnly struct {
	r io.Reader
}

func (o readOnly) Read(p []byte) (int, error) {
	return o.r.Read(p)
}

// writeOnly exposes only Write, so io.CopyBuffer uses the buffer it was handed.
//
// io.Copy and io.CopyBuffer both prefer a destination's ReadFrom when it has one, and
// ignore the buffer entirely in that case. *os.File has one, and for a source that is
// not a file it ends in a generic copy through a 32 KiB buffer allocated per call -
// which across a few hundred thousand entries is both the syscalls and the garbage that
// passing a buffer was meant to avoid.
type writeOnly struct {
	w io.Writer
}

func (o writeOnly) Write(p []byte) (int, error) {
	return o.w.Write(p)
}
