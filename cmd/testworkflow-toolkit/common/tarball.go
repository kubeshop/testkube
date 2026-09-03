package common

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
)

var (
	relativeCheckRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)
)

// WriteTarball packs files matching the patterns, resolved against dirPath.
func WriteTarball(stream io.Writer, dirPath string, files []string) error {
	return WriteTarballFrom(stream, dirPath, files, nil)
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
func WriteTarballFrom(stream io.Writer, dirPath string, files []string, mounts []string) error {
	// Ensure the absolute path
	if !filepath.IsAbs(dirPath) {
		var err error
		dirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return errors.Wrap(err, "failed to build absolute path for writing tarball")
		}
	}
	if len(mounts) == 0 {
		mounts = []string{dirPath}
	}

	// Prepare files archive
	gzipStream := gzip.NewWriter(stream)
	tarStream := tar.NewWriter(gzipStream)
	defer gzipStream.Close()
	defer tarStream.Close()

	// Append all the files
	walker, err := artifacts.CreateWalker(files, mounts, dirPath)
	if err != nil {
		return err
	}
	err = walker.Walk(os.DirFS("/"), func(path string, file fs.File, stat fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
			return nil
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

		// Copy the contents for regular files
		if !isSymlink {
			_, err = io.Copy(tarStream, file)
		}

		return err
	})
	return err
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

// normalizeEntryName turns a tar entry's name into the one slash-separated relative form
// that the rest of the extraction uses, or reports it as unsafe.
//
// Doing this in a single place is the point. A tar entry's name is always
// slash-separated by the format, but an archive can put anything in it, and deciding
// safety with host path rules while acting on a normalized string means the set of
// names that get through depends on which machine is unpacking.
func normalizeEntryName(raw string) (string, error) {
	unsafe := func() error {
		return fmt.Errorf("unsafe file path in the tarball: %s", raw)
	}

	name := filepath.ToSlash(raw)
	if name == "" {
		return "", unsafe()
	}

	// Absolute, judged on the normalized name rather than by the host's rules.
	if strings.HasPrefix(name, "/") {
		return "", unsafe()
	}

	// A volume or drive-qualified name is not a relative container path either, and
	// path.Join would silently treat it as one.
	if first, _, _ := strings.Cut(name, "/"); strings.Contains(first, ":") {
		return "", unsafe()
	}

	// Any parent traversal at all, which is stricter than the extraction strictly needs
	// - "a/../b" resolves inside the root - but keeps a name that reads as an escape
	// from being accepted on the grounds that it happens to cancel out.
	if relativeCheckRe.MatchString(name) {
		return "", unsafe()
	}

	cleaned := path.Clean(name)
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
		if !options.permits(path.Join(path.Clean(filepath.ToSlash(dirPath)), name)) {
			return fmt.Errorf("tarball entry %s is outside the paths this step asked to restore", name)
		}

		// Only the permission bits are honoured. header.Mode is attacker-controlled in
		// an archive from another execution, and nothing in a dependency tree needs
		// setuid, setgid or sticky.
		mode := os.FileMode(header.Mode).Perm() & 0o777

		switch header.Typeflag {
		case tar.TypeDir:
			err := root.MkdirAll(name, 0755)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory", filePath)
			}
		case tar.TypeReg:
			err := root.MkdirAll(path.Dir(name), 0755)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory tree", filePath)
			}
			outFile, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
			if err != nil {
				return errors.Wrapf(err, "%s: create file", filePath)
			}
			// header.Size is advisory - a tar header may understate what follows - so
			// the limit is enforced on what is actually read, not on what is declared.
			var copied int64
			if options.MaxTotalBytes > 0 {
				remaining := options.MaxTotalBytes - written
				copied, err = io.Copy(outFile, io.LimitReader(tarReader, remaining+1))
				if err == nil && copied > remaining {
					_ = outFile.Close()
					return fmt.Errorf("tarball expands to more than %d bytes", options.MaxTotalBytes)
				}
			} else {
				copied, err = io.Copy(outFile, tarReader)
			}
			if err != nil {
				_ = outFile.Close()
				return errors.Wrapf(err, "%s: write file", filePath)
			}
			written += copied
			_ = outFile.Close()
		case tar.TypeSymlink:
			err := root.MkdirAll(path.Dir(name), 0755)
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
