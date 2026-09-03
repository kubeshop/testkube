package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	initdata "github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	// cacheRetryMaxAttempts bounds transfer attempts. A failure past it is a miss, not
	// a failed step, so this only decides how hard the optimization tries.
	cacheRetryMaxAttempts = 5

	// cacheDefaultMaxSize caps the archive a save will upload. The tarball is written
	// to /tmp, which is an emptyDir counted against the pod's size limit, so an
	// unbounded cache could fill the volume and take the run down with it.
	cacheDefaultMaxSize = 5 << 30 // 5 GiB

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
	// Accepted and ignored: the processor passes the step's mounts to both stages, and a
	// restore writes wherever the archive says, confined by os.Root inside UnpackTarball.
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

// absoluteCachePath resolves a path against the step's working directory, matching how
// the processor decided which volume to mount for it.
func absoluteCachePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "/"
	}
	return filepath.Clean(filepath.Join(wd, path))
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
	if err := downloadCache(ctx, entry.URL); err != nil {
		// A half-unpacked node_modules is worse than none: an install would find some
		// of what it needs and skip the rest. Clear what was written and report a miss.
		clearCachePaths(spec.Paths, out)
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
			spec.Key, humanize.Bytes(uint64(entry.Size)), time.Since(started).Truncate(time.Millisecond))
	} else {
		fmt.Fprintf(out, "cache: partial hit for %q from %q (%s in %s)\n",
			spec.Key, entry.MatchedKey, humanize.Bytes(uint64(entry.Size)), time.Since(started).Truncate(time.Millisecond))
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

	archive, size, err := packCache(paths, mounts)
	if err != nil {
		return fmt.Errorf("packing the cache: %w", err)
	}
	defer func() {
		_ = archive.Close()
		_ = os.Remove(archive.Name())
	}()

	if maxSize > 0 && size > maxSize {
		fmt.Fprintf(out, "cache: not saving %q: the archive is %s, over the %s limit\n",
			key, humanize.Bytes(uint64(size)), humanize.Bytes(uint64(maxSize)))
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

	started := time.Now()
	if err := uploadCache(ctx, upload.URL, archive, size); err != nil {
		fmt.Fprintf(out, "cache: not saving %q: %s\n", key, err.Error())
		return nil
	}

	fmt.Fprintf(out, "cache: saved %q (%s in %s)\n",
		key, humanize.Bytes(uint64(size)), time.Since(started).Truncate(time.Millisecond))
	return nil
}

// packCache writes the cached paths into a temporary archive and returns it with its
// size, which a presigned PUT needs up front as a Content-Length.
//
// A real file rather than a streaming buffer, so that a retried upload can rewind.
func packCache(paths []string, mounts []string) (*os.File, int64, error) {
	file, err := os.CreateTemp(cacheTempDir(), "cache-*.tar.gz")
	if err != nil {
		return nil, 0, err
	}

	// Root at "/" with the absolute paths as patterns: cached paths may live in several
	// volumes at once and have no common ancestor below the root.
	if err := common.WriteTarballFrom(file, "/", paths, mounts); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, 0, err
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, 0, err
	}
	return file, stat.Size(), nil
}

func cacheTempDir() string {
	if dir := os.Getenv("TK_CACHE_TMP_DIR"); dir != "" {
		return dir
	}
	return os.TempDir()
}

// downloadCache fetches an entry and unpacks it at the root.
//
// The archive holds paths relative to "/", because a cache may span several volumes, so
// there is no single destination directory to confine it to. The confinement that
// matters is inside UnpackTarball, which resolves every entry through os.Root.
func downloadCache(ctx context.Context, url string) error {
	client := &http.Client{Timeout: cacheTransferTimeout}

	var lastErr error
	for attempt := 1; attempt <= cacheRetryMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("status code %d", resp.StatusCode)
			// An entry may have been evicted between the grant and the fetch, and a
			// grant may have expired. Neither is worth retrying.
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
				resp.Body.Close()
				return err
			}
		}
		if err == nil {
			err = common.UnpackTarball("/", resp.Body,
				common.WithMaxTotalBytes(cacheMaxUnpackedSize),
				common.WithMaxEntries(cacheMaxEntries))
			resp.Body.Close()
			if err == nil {
				return nil
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}
	return lastErr
}

func uploadCache(ctx context.Context, url string, archive *os.File, size int64) error {
	client := &http.Client{Timeout: cacheTransferTimeout}

	var lastErr error
	for attempt := 1; attempt <= cacheRetryMaxAttempts; attempt++ {
		if _, err := archive.Seek(0, io.SeekStart); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, archive)
		if err != nil {
			return err
		}
		req.ContentLength = size
		req.Header.Set("Content-Type", "application/gzip")

		resp, err := client.Do(req)
		if err == nil {
			status := resp.StatusCode
			resp.Body.Close()
			if status >= 200 && status < 300 {
				return nil
			}
			err = fmt.Errorf("status code %d", status)
			// A rejected grant will be rejected again.
			if status == http.StatusForbidden || status == http.StatusBadRequest {
				return err
			}
		}
		lastErr = err
	}
	return lastErr
}

// clearCachePaths empties the cached directories without removing them, because an
// auto-mounted path is a mount point and cannot be unlinked.
func clearCachePaths(paths []string, out io.Writer) {
	for _, path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
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
