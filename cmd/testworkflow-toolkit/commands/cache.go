package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	initdata "github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	// cacheRetryMaxAttempts bounds transfer attempts. A failure past it is a miss, not
	// a failed step, so this only decides how hard the optimization tries.
	cacheRetryMaxAttempts = 5

	// cacheDefaultMaxSize caps the archive a save will write. The processor sizes the
	// staging volume from the same constant, so the two cannot drift into a limit the
	// volume is too small to hold.
	cacheDefaultMaxSize = executioncache.MaxArchiveSize

	// cacheMaxUnpackedSize and cacheMaxEntries bound what a restore may expand into.
	// The archive was written by an earlier - possibly different - workflow, so its
	// contents are not this execution's own doing.
	cacheMaxUnpackedSize = 10 << 30 // 10 GiB
	cacheMaxEntries      = 500_000

	cacheTransferTimeout = 30 * time.Minute
)

// NewCacheCmd restores and saves a step's dependency cache.
//
// Neither subcommand ever exits non-zero. A cache is an optimization: a miss, an
// unreachable control plane, a corrupt archive or a refused upload all leave the step to
// install from the network exactly as it would have without any cache, and turning any
// of that into a failure would make caching strictly worse than not caching.
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Restore and save a step's dependency cache",
	}
	cmd.AddCommand(newCacheRestoreCmd())
	cmd.AddCommand(newCacheSaveCmd())
	return cmd
}

func newCacheRestoreCmd() *cobra.Command {
	var encoded string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a dependency cache before a step runs",

		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			if err := runCacheRestore(cmd.Context(), encoded, newCacheRepository(), out); err != nil {
				// Never a non-zero exit: see NewCacheCmd.
				fmt.Fprintf(out, "cache: not restored: %s\n", err.Error())
			}
		},
	}

	cmd.Flags().StringVar(&encoded, "base64", "", "base64-encoded cache specification")
	// Accepted and ignored: the processor passes the step's mounts to both stages, but a
	// restore is confined to the paths the step declared in its cache block, which is
	// narrower than its volumes - see downloadCache.
	cmd.Flags().StringArrayP("mount", "m", nil, "volume roots (unused by restore)")
	return cmd
}

func newCacheSaveCmd() *cobra.Command {
	var encoded string
	var mounts []string
	var statePath string
	var maxSize int64

	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save a dependency cache after a step passes",

		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			if statePath == "" {
				statePath = os.Getenv("TK_CACHE_STATE")
			}
			if err := runCacheSave(cmd.Context(), encoded, mounts, statePath, maxSize, newCacheRepository(), out); err != nil {
				fmt.Fprintf(out, "cache: not saved: %s\n", err.Error())
			}
		},
	}

	cmd.Flags().StringVar(&encoded, "base64", "", "base64-encoded cache specification")
	cmd.Flags().StringArrayVarP(&mounts, "mount", "m", nil, "volume roots that may be read")
	cmd.Flags().StringVar(&statePath, "state", "", "path the restore stage recorded its key in")
	cmd.Flags().Int64Var(&maxSize, "max-size", cacheDefaultMaxSize, "largest archive to upload, in bytes")
	return cmd
}

// resolveCacheSpec decodes the payload and resolves its expressions in the pod.
//
// This is where a key like 'npm-{{ hash_files("package-lock.json") }}' actually becomes
// a value: the machine carries the filesystem functions and the working directory is
// already the step's, because testworkflow-init chdirs before running the command.
func resolveCacheSpec(encoded string) (spec executioncache.Args, resolved executioncache.Args, err error) {
	if encoded == "" {
		return spec, resolved, fmt.Errorf("no cache specification provided")
	}
	if err := expressions.DecodeBase64JSON(encoded, &spec); err != nil {
		return spec, resolved, fmt.Errorf("reading the cache specification: %w", err)
	}

	machine := initdata.GetBaseTestWorkflowMachine()

	resolved = executioncache.Args{Scope: spec.Scope, State: spec.State}

	key, err := expressions.CompileAndResolveTemplate(spec.Key, machine, expressions.FinalizerFail)
	if err != nil {
		return spec, resolved, fmt.Errorf("resolving the cache key: %w", err)
	}
	resolved.Key, _ = key.Static().StringValue()

	if err := checkKeyComponents(spec.Key, machine); err != nil {
		return spec, resolved, err
	}

	for _, restoreKey := range spec.RestoreKeys {
		value, err := expressions.CompileAndResolveTemplate(restoreKey, machine, expressions.FinalizerFail)
		if err != nil {
			// One unusable fallback should not discard the others.
			continue
		}
		if str, _ := value.Static().StringValue(); str != "" {
			resolved.RestoreKeys = append(resolved.RestoreKeys, str)
		}
	}

	for _, path := range spec.Paths {
		value, err := expressions.CompileAndResolveTemplate(path, machine, expressions.FinalizerFail)
		if err != nil {
			return spec, resolved, fmt.Errorf("resolving the cache path %q: %w", path, err)
		}
		str, _ := value.Static().StringValue()
		if str == "" {
			continue
		}
		resolved.Paths = append(resolved.Paths, absoluteCachePath(str))
	}

	return spec, resolved, nil
}

// checkKeyComponents refuses a key in which some part of the template evaluated to
// nothing.
//
// ValidateKey catches a key that is empty outright, which is what an unmatched
// hash_files() produces on its own. It does not catch the far commoner shape: a key of
// npm-v3-{{ hash_files("package-lock.json") }} with the lockfile absent resolves to the
// perfectly valid-looking "npm-v3-", and every step in that state then shares one entry.
//
// That is worse than the empty key it slips past, not better, because it looks like a
// working cache. Three things compound it. The degenerate key sits exactly where its own
// restoreKeys prefix points, so the entry is a candidate for every lookup under that
// prefix and not only for the runs that produced it - and the most recently saved match
// wins, so one such save can become the preferred fallback for runs that do have a
// lockfile. Entries are immutable, so whatever the first such run stored is what the key
// holds for its lifetime. And under scope: environment it crosses workflows.
//
// The rule is any empty component, not specifically an empty hash_files(), because what
// matters is that every part of a key distinguishes something: npm-{{ hash_files("a")
// }}-{{ hash_files("b") }} with b absent silently stops distinguishing b's state, and an
// empty config value does the same. It applies to the key alone and deliberately not to
// restoreKeys - a restore key is a prefix, and the whole defect here is a key that has
// become indistinguishable from one.
//
// Refusing is a miss, not a failure: the caller reports it and the step installs from the
// network exactly as it would with no cache configured. hash_files() returns "" rather
// than an error precisely so the caller can make this decision, which until now it only
// half made.
func checkKeyComponents(template string, machine expressions.Machine) error {
	parts, err := expressions.TemplateExpressions(template)
	if err != nil {
		// The key already resolved, so this cannot be a syntax error the caller has not
		// seen. Nothing to add.
		return nil
	}

	for _, part := range parts {
		value, err := expressions.CompileAndResolve(part, machine, expressions.FinalizerFail)
		if err != nil {
			// Same reasoning: resolving the whole key succeeded, so a part that will not
			// resolve on its own is this check's problem and not the key's.
			continue
		}
		if str, _ := value.Static().StringValue(); str == "" {
			return fmt.Errorf(
				"cache key %q: %s evaluated to nothing, so the key no longer identifies what it was derived from - every run in this state would share one entry",
				template, part)
		}
	}

	return nil
}

// absoluteCachePath resolves a path against the step's working directory, matching how
// the processor decided which volume to mount for it.
// Resolved with `path`, not `filepath`: a cache path names a directory inside a Linux
// container, and it has to agree byte for byte with the allowlist the unpacker checks
// entries against, which is expressed the same way.
func absoluteCachePath(declared string) string {
	if strings.HasPrefix(declared, "/") {
		return path.Clean(declared)
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "/"
	}
	return path.Clean(path.Join(filepath.ToSlash(wd), declared))
}

func runCacheRestore(ctx context.Context, encoded string, repository executioncache.Repository, out io.Writer) error {
	_, spec, err := resolveCacheSpec(encoded)
	state := executioncache.State{Hit: executioncache.HitMiss, Scope: spec.Scope}
	// The state file is written on every path, including the failures, so that the save
	// stage can tell "restore decided not to" from "restore never ran".
	defer func() {
		state.Key = spec.Key
		state.Paths = spec.Paths
		writeCacheState(spec.State, state, out)
	}()

	if err != nil {
		return err
	}
	if err := executioncache.ValidateKey(spec.Key); err != nil {
		// Most often an unmatched hash_files(), which yields "". Caching every such
		// step under one shared entry would be worse than not caching at all.
		return err
	}
	if reason := executioncache.Reason(repository); reason != "" {
		fmt.Fprintf(out, "cache: miss for %q: %s\n", spec.Key, reason)
		return nil
	}

	entry, err := repository.Restore(ctx, executioncache.RestoreRequest{
		Key:         spec.Key,
		RestoreKeys: spec.RestoreKeys,
		Scope:       executioncache.ParseScope(spec.Scope),
	})
	if err != nil {
		if reason, degraded := executioncache.Degraded(err); degraded {
			fmt.Fprintf(out, "cache: miss for %q: %s\n", spec.Key, reason)
			return nil
		}
		return err
	}
	if !entry.Hit {
		fmt.Fprintf(out, "cache: miss for %q\n", spec.Key)
		return nil
	}

	started := time.Now()
	restored := entry.Size
	// Closed explicitly: the store holds an open handle on the mount, and a restore is
	// one of several stages sharing a container.
	store := cacheStore(out)
	defer store.Close()

	wrote, size, err := downloadCache(ctx, store, entry.URL, spec.Paths)
	if size > 0 {
		// The object was a pointer, so its own size says nothing about what was
		// restored; the pointer carries what the entry holds.
		restored = size
	}
	if err != nil {
		// A half-unpacked node_modules is worse than none: an install would find some
		// of what it needs and skip the rest. Clear what was written and report a miss.
		//
		// Only when something actually was written, though. A failure that wrote
		// nothing - an unreachable store, an expired grant, a 404 for an entry that
		// expired between the grant and the fetch, a body that is not a gzip stream at
		// all, a first entry rejected for landing outside the declared paths - has
		// nothing to undo. And the declared paths are not necessarily empty to begin
		// with: one under `mount: false` lives in a volume somebody else populated, and
		// one inside the repository checkout holds the checkout. Clearing on those
		// turned a cache miss, which is meant to leave the step exactly as it would have
		// been with no cache configured, into deleting the step's own inputs.
		if wrote {
			clearCachePaths(spec.Paths, out)
		}
		fmt.Fprintf(out, "cache: miss for %q: could not restore the entry: %s\n", spec.Key, err.Error())
		return nil
	}

	state.Hit = executioncache.HitExact
	if !entry.Exact {
		state.Hit = executioncache.HitPartial
	}
	state.MatchedKey = entry.MatchedKey

	if entry.Exact {
		fmt.Fprintf(out, "cache: hit for %q (%s in %s)\n",
			spec.Key, humanize.Bytes(uint64(restored)), time.Since(started).Truncate(time.Millisecond))
	} else {
		fmt.Fprintf(out, "cache: partial hit for %q from %q (%s in %s)\n",
			spec.Key, entry.MatchedKey, humanize.Bytes(uint64(restored)), time.Since(started).Truncate(time.Millisecond))
	}
	return nil
}

func runCacheSave(ctx context.Context, encoded string, mounts []string, statePath string, maxSize int64, repository executioncache.Repository, out io.Writer) error {
	_, spec, err := resolveCacheSpec(encoded)
	if err != nil {
		return err
	}

	key := spec.Key
	paths := spec.Paths

	// Prefer what the restore stage resolved. An install may have rewritten the
	// lockfile the key hashes, so recomputing it here could store the entry under a key
	// no later run will look for.
	if state, ok := readCacheState(statePath); ok {
		if state.Hit == executioncache.HitExact {
			fmt.Fprintf(out, "cache: hit for %q, nothing to save\n", state.Key)
			return nil
		}
		if state.Key != "" {
			key = state.Key
		}
		if len(state.Paths) > 0 {
			paths = state.Paths
		}
	}

	if err := executioncache.ValidateKey(key); err != nil {
		return err
	}
	if reason := executioncache.Reason(repository); reason != "" {
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, reason)
		return nil
	}

	// The shared volume replaces the archive entirely: the tree is copied to it and the
	// object holds a pointer. Kept in its own function so the object-store path below
	// is exactly what it was, which is what a pod with no volume - or one whose volume
	// turned out to be unusable - still runs.
	if inbox := cacheInbox(out); inbox != nil {
		defer inbox.Close()
		return saveToVolume(ctx, inbox, key, spec.Scope, paths, maxSize, repository, out)
	}

	// Timed from here, not from the upload. Packing reads and compresses the whole tree
	// and is usually the larger half of a save, so timing only the transfer reported a
	// number that had little to do with how long the step waited.
	started := time.Now()

	archive, size, entries, err := packCache(paths, mounts, maxSize)
	if err != nil {
		if errors.Is(err, errArchiveTooLarge) {
			fmt.Fprintf(out, "cache: not saving %q: the archive is over the %s limit\n",
				key, humanize.Bytes(uint64(maxSize)))
			return nil
		}
		return fmt.Errorf("packing the cache: %w", err)
	}
	packed := time.Since(started)
	defer func() {
		_ = archive.Close()
		_ = os.Remove(archive.Name())
	}()

	// Storing nothing under a key is worse than storing nothing at all: an entry is
	// immutable for its lifetime, so an empty archive would answer every later run with
	// a hit that restores nothing, and no rerun could displace it. Nothing to pack means
	// the paths were never created - the install put its output somewhere else, or the
	// step did not need to install - and either way there is nothing to cache.
	if entries == 0 {
		fmt.Fprintf(out, "cache: not saving %q: nothing was found under %s\n", key, strings.Join(paths, ", "))
		return nil
	}

	upload, err := repository.Save(ctx, executioncache.SaveRequest{
		Key:   key,
		Scope: executioncache.ParseScope(spec.Scope),
		Size:  size,
	})
	if err != nil {
		if reason, degraded := executioncache.Degraded(err); degraded {
			fmt.Fprintf(out, "cache: not saving %q: %s\n", key, reason)
			return nil
		}
		return err
	}
	if upload.AlreadyExists {
		// Another execution stored this key between our restore and now. Entries are
		// immutable, so there is nothing to do and nothing wrong.
		fmt.Fprintf(out, "cache: %q is already stored, nothing to save\n", key)
		return nil
	}

	uploadStarted := time.Now()
	if err := uploadCache(ctx, upload.URL, upload.Headers, archive, size); err != nil {
		if errors.Is(err, errCacheEntryWon) {
			// Losing the race is not a failure. Both executions reached the same
			// content-derived key, so the entry now stored is the one this step would
			// have written, and the next run will hit it.
			fmt.Fprintf(out, "cache: %q was stored by another execution first, nothing to save\n", key)
			return nil
		}
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}

	// Both halves are reported separately: they scale with different things - packing
	// with the size and file count of the tree, the upload with the archive and the
	// link - so one total hides which of them to do something about.
	fmt.Fprintf(out, "cache: saved %q (%s in %s, %s packing and %s uploading)\n",
		key, humanize.Bytes(uint64(size)), time.Since(started).Truncate(time.Millisecond),
		packed.Truncate(time.Millisecond), time.Since(uploadStarted).Truncate(time.Millisecond))
	return nil
}

// cachePackPatterns turns the cached paths into the patterns the walker matches against.
//
// A cached path names a directory, but the walker tests every candidate against the
// patterns with doublestar and skips directories - and "/data/node_modules" matches only
// the directory itself, so a bare path matches nothing whatsoever. The archive then comes
// out empty, the save succeeds, and the next run gets a hit that restores nothing: the
// cache appears to work and quietly does not.
//
// Appending "/**" is what makes the contents match. The bare path is kept alongside it
// because "/x/**" does not match "/x", and a cached path is allowed to be a single file.
func cachePackPatterns(paths []string) []string {
	patterns := make([]string, 0, len(paths)*2)
	for _, declared := range paths {
		patterns = append(patterns, declared, path.Join(declared, "**"))
	}
	return patterns
}

// packCache writes the cached paths into a temporary archive and returns it with its
// size, which a presigned PUT needs up front as a Content-Length, and the number of
// entries it holds.
//
// A real file rather than a streaming buffer, so that a retried upload can rewind.
// errArchiveTooLarge stops a pack that has reached the size limit.
//
// The limit is enforced while writing rather than checked afterwards, because the
// archive is staged on a volume sized from the same limit: writing the whole thing and
// then refusing it would overrun that volume first, and exceeding a volume's size limit
// evicts the pod. A cache that cannot be stored has to skip, not take the step down.
var errArchiveTooLarge = errors.New("archive is over the size limit")

// boundedWriter fails the write once more than limit bytes have gone through it.
type boundedWriter struct {
	w       io.Writer
	limit   int64
	written int64
}

func (b *boundedWriter) Write(p []byte) (int, error) {
	if b.limit > 0 && b.written+int64(len(p)) > b.limit {
		return 0, errArchiveTooLarge
	}
	n, err := b.w.Write(p)
	b.written += int64(n)
	return n, err
}

func packCache(paths []string, mounts []string, maxSize int64) (file *os.File, size int64, entries int, err error) {
	file, err = os.CreateTemp(cacheTempDir(), "cache-*.tar.gz")
	if err != nil {
		return nil, 0, 0, err
	}
	discard := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}

	// Root at "/" with the absolute paths as patterns: cached paths may live in several
	// volumes at once and have no common ancestor below the root.
	entries, err = common.WriteTarballFrom(&boundedWriter{w: file, limit: maxSize}, "/", cachePackPatterns(paths), mounts)
	if err != nil {
		discard()
		return nil, 0, 0, err
	}

	stat, err := file.Stat()
	if err != nil {
		discard()
		return nil, 0, 0, err
	}
	return file, stat.Size(), entries, nil
}

func cacheTempDir() string {
	if dir := os.Getenv("TK_CACHE_TMP_DIR"); dir != "" {
		return dir
	}
	return os.TempDir()
}

// cacheStore opens the shared cache volume for reading, or returns nil.
//
// Nil is not a failure: an entry stored as an archive restores without it, and one
// stored as a pointer reports a miss that names the reason. The reason is printed here
// rather than swallowed because a pod that was meant to have the volume and does not is
// an operator's problem, and an unexplained cold cache would be the only symptom.
func cacheStore(out io.Writer) *volume.Store {
	mountPath := os.Getenv(volume.EnvStorePath)
	if mountPath == "" {
		return nil
	}
	store, reason := volume.OpenStore(mountPath)
	if store == nil {
		fmt.Fprintf(out, "cache: not using the shared volume: %s\n", reason)
	}
	return store
}

// cacheInbox opens this execution's own directory on the shared cache volume, or
// returns nil to save to the object store instead.
func cacheInbox(out io.Writer) *volume.Inbox {
	mountPath := os.Getenv(volume.EnvInboxPath)
	if mountPath == "" {
		return nil
	}
	inbox, reason := volume.OpenInbox(mountPath, os.Getenv(volume.EnvInboxName))
	if inbox == nil {
		fmt.Fprintf(out, "cache: not using the shared volume: %s\n", reason)
	}
	return inbox
}

// downloadCache fetches an entry and unpacks it at the root.
//
// The archive holds paths relative to "/", because a cache may span several volumes and
// so has no single destination directory to extract into. That makes os.Root's
// confinement to "/" no confinement at all, so the paths this step asked to restore are
// passed as an allowlist: an archive is written by whoever populated the cache, and
// under an environment-scoped cache that is another workflow, which must not be able to
// write anywhere else in this container.
//
// The step's own mount roots would be a weaker allowlist - they cover every volume the
// step has, including the repository checkout - so the declared cache paths are used
// instead. A rejection is treated like any other failed restore: the paths are cleared
// and the step reports a miss, rather than running against a tree that was filtered.
// wrote reports whether extraction created anything, which is what tells the caller
// whether a failure left something to clean up. It is not "extraction was attempted": a
// transfer can succeed and the archive still be rejected before the first entry is
// created. It is sticky across retries, because an attempt that wrote part of a tree
// leaves that tree behind even if a later attempt fails earlier.
// size reports how much the entry held, for the line the restore prints. It is zero
// for an archive, whose compressed size the caller already knows from the grant.
func downloadCache(ctx context.Context, store *volume.Store, url string, allowedPaths []string) (wrote bool, size int64, err error) {
	client := &http.Client{Timeout: cacheTransferTimeout}

	var lastErr error
	for attempt := 1; attempt <= cacheRetryMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return wrote, 0, err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("status code %d", resp.StatusCode)
			// An entry may have been evicted between the grant and the fetch, and a
			// grant may have expired. Neither is worth retrying.
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
				resp.Body.Close()
				return wrote, 0, err
			}
		}
		if err == nil {
			// An object either points at an entry on the shared volume or is the
			// archive itself. Which one it is, is decided by its first bytes rather
			// than by configuration, so an entry written before the volume existed -
			// or by a cluster that has none - restores with nothing to get wrong.
			body := bufio.NewReaderSize(resp.Body, volume.MaxPointerBytes)
			head, peekErr := body.Peek(len(volume.Magic))
			if peekErr == nil && bytes.Equal(head, []byte(volume.Magic)) {
				wrote, size, err = restoreFromVolume(store, body, allowedPaths)
				resp.Body.Close()
				// A pointer that cannot be followed will not start working on a retry:
				// the bytes only ever existed on the volume.
				return wrote, size, err
			}

			err = common.UnpackTarball("/", body,
				common.WithMaxTotalBytes(cacheMaxUnpackedSize),
				common.WithMaxEntries(cacheMaxEntries),
				common.WithAllowedRoots(allowedPaths...),
				common.WithWriteObserver(func() { wrote = true }))
			resp.Body.Close()
			if err == nil {
				return wrote, 0, nil
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}
	return wrote, 0, lastErr
}

// restoreFromVolume copies an entry off the shared volume onto the filesystem.
//
// Every failure here is a miss rather than an error the step sees, which is the same
// contract the archive path has - but the reasons are worth naming separately, because
// "the pointer is there and the entry is not" is a retention misconfiguration, and a
// silent miss would hide it behind a cold cache.
func restoreFromVolume(store *volume.Store, body io.Reader, allowedPaths []string) (bool, int64, error) {
	head, err := io.ReadAll(io.LimitReader(body, volume.MaxPointerBytes))
	if err != nil {
		return false, 0, fmt.Errorf("reading the cache pointer: %w", err)
	}
	pointer, ok := volume.Decode(head)
	if !ok {
		return false, 0, errors.New("the entry is a cache pointer this agent cannot read")
	}
	if store == nil {
		return false, 0, errors.New("the entry is on a shared cache volume this pod has not mounted")
	}

	root, err := store.OpenEntry(pointer)
	if err != nil {
		return false, 0, fmt.Errorf("opening the cache entry on the shared volume: %w", err)
	}
	defer root.Close()

	wrote, err := volume.RestoreTree(root, allowedPaths, volume.CopyLimits{
		MaxTotalBytes: cacheMaxUnpackedSize,
		MaxEntries:    cacheMaxEntries,
	})
	if err != nil {
		return wrote, pointer.Size, err
	}
	return wrote, pointer.Size, nil
}

// errCacheEntryWon reports that another execution stored this key first.
//
// The control plane signs the upload with a condition - only if the object is absent -
// so of two executions racing on one key exactly one upload is applied and the other is
// refused. That is the mechanism keeping a stored entry immutable, and being the loser
// is a normal outcome: the winner stored an equivalent tree under the same
// content-derived key.
var errCacheEntryWon = errors.New("another execution stored this key first")

// uploadCache sends the archive, with whatever headers the grant requires.
//
// The headers are covered by the signature, so they are not optional decoration: an
// upload that omits them is rejected, which is deliberate - it means the condition
// cannot be dropped to turn the request back into a plain overwrite.
func uploadCache(ctx context.Context, url string, headers map[string]string, archive io.ReadSeeker, size int64) error {
	client := &http.Client{Timeout: cacheTransferTimeout}

	var lastErr error
	for attempt := 1; attempt <= cacheRetryMaxAttempts; attempt++ {
		if _, err := archive.Seek(0, io.SeekStart); err != nil {
			if lastErr != nil {
				// Whatever stopped the previous attempt is the useful half. Returning
				// the rewind failure on its own reported a symptom of retrying and hid
				// the reason there was anything to retry.
				return fmt.Errorf("%w (and the archive could not be rewound to retry: %s)", lastErr, err)
			}
			return err
		}
		// io.NopCloser, because http.Client.Do closes the request body and the body here
		// is the archive itself - so a plain *os.File left every attempt after the first
		// to fail on the rewind above with "file already closed", whatever the store had
		// actually answered.
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, io.NopCloser(archive))
		if err != nil {
			return err
		}
		req.ContentLength = size
		req.Header.Set("Content-Type", "application/gzip")
		for name, value := range headers {
			req.Header.Set(name, value)
		}

		resp, err := client.Do(req)
		if err == nil {
			status := resp.StatusCode
			resp.Body.Close()
			if status >= 200 && status < 300 {
				return nil
			}
			// The condition refused the write, so the entry is already there. Retrying
			// would only be refused again.
			if executioncache.UploadRefused(status) {
				return errCacheEntryWon
			}
			err = fmt.Errorf("status code %d", status)
			// None of these change on a second attempt: the first two are a grant the
			// store will not honour, and the others are the store saying it does not
			// implement what the request asked of it - most likely the condition that
			// makes the write conditional at all.
			switch status {
			case http.StatusForbidden, http.StatusBadRequest,
				http.StatusNotImplemented, http.StatusMethodNotAllowed:
				return err
			}
		}
		lastErr = err
	}
	return lastErr
}

// clearCachePaths empties the cached directories without removing them, because an
// auto-mounted path is a mount point and cannot be unlinked.
//
// A cached path that is a plain file is removed outright instead. Nothing stops a
// workflow naming one, and it is the case that matters most here: os.ReadDir answers
// ENOTDIR for it, so treating a read failure as "nothing to clean" left the restored
// file in place. The step would then have gone on to consume cache content after the
// restore had already reported a miss - the one outcome the cleanup exists to prevent,
// reached by the one path that never looked like a failure.
func clearCachePaths(paths []string, out io.Writer) {
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			// Absent, which is nothing to clean rather than a problem: a restore that
			// failed before creating anything is the ordinary case.
			continue
		}

		if !info.IsDir() {
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(out, "cache: could not remove %s after a failed restore: %s\n", path, err.Error())
			}
			continue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			fmt.Fprintf(out, "cache: could not read %s to clean it after a failed restore: %s\n", path, err.Error())
			continue
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
				fmt.Fprintf(out, "cache: could not clean %s after a failed restore: %s\n", path, err.Error())
			}
		}
	}
}

func writeCacheState(path string, state executioncache.State, out io.Writer) {
	if path == "" {
		return
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		fmt.Fprintf(out, "cache: could not record the cache state: %s\n", err.Error())
		return
	}
	// Group-writable, and so is the directory: the stages may run as different users.
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		fmt.Fprintf(out, "cache: could not record the cache state: %s\n", err.Error())
		return
	}
	if err := os.WriteFile(path, encoded, 0o666); err != nil {
		fmt.Fprintf(out, "cache: could not record the cache state: %s\n", err.Error())
	}
}

func readCacheState(path string) (executioncache.State, bool) {
	if path == "" {
		return executioncache.State{}, false
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return executioncache.State{}, false
	}
	var state executioncache.State
	if err := json.Unmarshal(contents, &state); err != nil {
		return executioncache.State{}, false
	}
	return state, true
}

// saveToVolume copies the cached tree onto the shared volume and stores a pointer to it.
//
// The ordering is what keeps an entry from ever being half-published, and it is the
// same ordering the archive path has, with the copy standing in for the pack:
//
//	copy    - into a staging directory under a name no reader follows
//	Save    - the control plane decides whether this execution may store this key
//	Commit  - rename, so the entry appears whole or not at all
//	upload  - the pointer, conditionally, which is what resolves a race on one key
//
// Everything that does not end in a published pointer discards what it staged. An entry
// is reachable only through its pointer, so one left behind is invisible until the
// volume fills, and it can be gigabytes.
func saveToVolume(
	ctx context.Context,
	inbox *volume.Inbox,
	key, scope string,
	paths []string,
	maxSize int64,
	repository executioncache.Repository,
	out io.Writer,
) error {
	started := time.Now()

	dir, staged, err := inbox.Stage()
	if err != nil {
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}
	committed := false
	defer func() {
		if !committed {
			inbox.DiscardStaged(staged)
		}
	}()

	size, entries, err := volume.SaveTree(dir, paths, volume.CopyLimits{
		MaxTotalBytes: maxSize,
		MaxEntries:    cacheMaxEntries,
	})
	if err != nil {
		if errors.Is(err, volume.ErrTooLarge) {
			fmt.Fprintf(out, "cache: not saving %q: the entry is over the %s limit\n",
				key, humanize.Bytes(uint64(maxSize)))
			return nil
		}
		// A volume that filled, or went away, mid-copy. The step installed what it
		// needed either way, so this is a save that did not happen rather than a
		// failure the step has to care about.
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}
	copied := time.Since(started)

	// Storing nothing under a key is worse than storing nothing at all: an entry is
	// immutable for its lifetime, so an empty one would answer every later run with a
	// hit that restores nothing, and no rerun could displace it.
	if entries == 0 {
		fmt.Fprintf(out, "cache: not saving %q: nothing was found under %s\n", key, strings.Join(paths, ", "))
		return nil
	}

	upload, err := repository.Save(ctx, executioncache.SaveRequest{
		Key:   key,
		Scope: executioncache.ParseScope(scope),
		Size:  size,
	})
	if err != nil {
		if reason, degraded := executioncache.Degraded(err); degraded {
			fmt.Fprintf(out, "cache: not saving %q: %s\n", key, reason)
			return nil
		}
		return err
	}
	if upload.AlreadyExists {
		fmt.Fprintf(out, "cache: %q is already stored, nothing to save\n", key)
		return nil
	}

	pointer, err := inbox.Commit(staged, size)
	if err != nil {
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}
	committed = true
	published := false
	defer func() {
		if !published {
			inbox.Discard(pointer)
		}
	}()

	uploadStarted := time.Now()
	body := bytes.NewReader(volume.Encode(pointer))
	if err := uploadCache(ctx, upload.URL, upload.Headers, body, int64(body.Len())); err != nil {
		if errors.Is(err, errCacheEntryWon) {
			fmt.Fprintf(out, "cache: %q was stored by another execution first, nothing to save\n", key)
			return nil
		}
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}
	published = true

	// Both halves are reported separately for the same reason the archive path reports
	// them separately: the copy scales with the tree, the upload with the control
	// plane, and one total hides which of them to do something about.
	fmt.Fprintf(out, "cache: saved %q to the shared volume (%s in %s, %s copying and %s storing)\n",
		key, humanize.Bytes(uint64(size)), time.Since(started).Truncate(time.Millisecond),
		copied.Truncate(time.Millisecond), time.Since(uploadStarted).Truncate(time.Millisecond))
	return nil
}
