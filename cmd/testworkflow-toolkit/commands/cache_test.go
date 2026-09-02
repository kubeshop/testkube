package commands

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// fakeCacheRepository stands in for the control plane.
type fakeCacheRepository struct {
	restore    executioncache.RestoreResult
	restoreErr error
	save       executioncache.SaveResult
	saveErr    error

	restoreCalls int
	saveCalls    int
	savedKey     string
	savedSize    int64
}

func (f *fakeCacheRepository) Restore(context.Context, executioncache.RestoreRequest) (executioncache.RestoreResult, error) {
	f.restoreCalls++
	return f.restore, f.restoreErr
}

func (f *fakeCacheRepository) Save(_ context.Context, req executioncache.SaveRequest) (executioncache.SaveResult, error) {
	f.saveCalls++
	f.savedKey = req.Key
	f.savedSize = req.Size
	return f.save, f.saveErr
}

func encodeCacheArgs(t *testing.T, args executioncache.Args) string {
	t.Helper()
	encoded, err := expressions.EncodeBase64JSON(args)
	require.NoError(t, err)
	return encoded
}

// cacheTarball builds an archive holding one file at the given root-relative path.
func cacheTarball(t *testing.T, name, contents string) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(contents)),
	}))
	_, err := tw.Write([]byte(contents))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// TestRunCacheRestore_DegradesToMiss is the contract that matters most: nothing the
// control plane or the network does may turn into a failed step.
func TestRunCacheRestore_DegradesToMiss(t *testing.T) {
	cases := []struct {
		name       string
		repository executioncache.Repository
		wantOutput string
	}{
		{
			name:       "capability absent",
			repository: executioncache.Unsupported("no capability here"),
			wantOutput: "no capability here",
		},
		{
			name: "control plane does not implement the method",
			repository: &fakeCacheRepository{
				restoreErr: status.Error(codes.Unimplemented, "unknown method"),
			},
			wantOutput: "cannot serve dependency caches",
		},
		{
			name: "control plane refuses the request",
			repository: &fakeCacheRepository{
				restoreErr: status.Error(codes.ResourceExhausted, "over quota"),
			},
			wantOutput: "over quota",
		},
		{
			name:       "plain miss",
			repository: &fakeCacheRepository{restore: executioncache.RestoreResult{Hit: false}},
			wantOutput: "cache: miss",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			statePath := filepath.Join(t.TempDir(), "state.json")

			err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
				Key:   "npm-abc",
				Paths: []string{"/tmp/does-not-matter"},
				State: statePath,
			}), tc.repository, out)

			assert.NoError(t, err, "a cache problem must never surface as an error")
			assert.Contains(t, out.String(), tc.wantOutput)

			// A miss still records state, so the save stage can tell it apart from a
			// restore that never ran at all.
			state := readState(t, statePath)
			assert.Equal(t, executioncache.HitMiss, state.Hit)
		})
	}
}

// TestRunCacheRestore_UnresolvableKeyIsNotFatal covers the case the base64 payload
// exists for: a key hashing a lockfile that is not there.
func TestRunCacheRestore_UnresolvableKeyIsNotFatal(t *testing.T) {
	repository := &fakeCacheRepository{}
	out := &bytes.Buffer{}
	statePath := filepath.Join(t.TempDir(), "state.json")

	// hash_files over nothing yields "", which ValidateKey refuses - caching every
	// such step under one shared entry would be worse than not caching.
	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   `{{ hash_files("definitely-absent-*.lock") }}`,
		Paths: []string{"/tmp/x"},
		State: statePath,
	}), repository, out)

	assert.Error(t, err, "the caller logs this without exiting; see NewCacheCmd")
	assert.Zero(t, repository.restoreCalls, "an unusable key must not reach the control plane")
	assert.Equal(t, executioncache.HitMiss, readState(t, statePath).Hit)
}

func TestRunCacheRestore_Hit(t *testing.T) {
	root := t.TempDir()
	// Restores unpack at "/", so point the archive at a path inside the temp dir.
	relative := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	archive := cacheTarball(t, filepath.ToSlash(filepath.Join(relative, "restored.txt"))[1:], "from-cache")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{root},
		State: statePath,
	}), &fakeCacheRepository{
		restore: executioncache.RestoreResult{
			Hit: true, Exact: true, MatchedKey: "npm-abc", URL: server.URL, Size: int64(len(archive)),
		},
	}, out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "cache: hit")

	state := readState(t, statePath)
	assert.Equal(t, executioncache.HitExact, state.Hit)
	assert.Equal(t, "npm-abc", state.MatchedKey)
}

func TestRunCacheRestore_PartialHitIsRecordedSoSaveStillRuns(t *testing.T) {
	root := t.TempDir()
	relative := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	archive := cacheTarball(t, filepath.ToSlash(filepath.Join(relative, "restored.txt"))[1:], "older")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	require.NoError(t, runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:         "npm-new",
		RestoreKeys: []string{"npm-"},
		Paths:       []string{root},
		State:       statePath,
	}), &fakeCacheRepository{
		restore: executioncache.RestoreResult{
			Hit: true, Exact: false, MatchedKey: "npm-old", URL: server.URL, Size: int64(len(archive)),
		},
	}, out))

	assert.Contains(t, out.String(), "partial hit")
	assert.Equal(t, executioncache.HitPartial, readState(t, statePath).Hit)
}

// TestRunCacheRestore_CorruptArchiveClearsPartialState: a half-restored tree fed to an
// installer is worse than an empty one, because the install would skip what is present.
func TestRunCacheRestore_CorruptArchiveClearsPartialState(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "stale.txt"), []byte("x"), 0o644))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("this is not a gzip stream"))
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{root},
		State: statePath,
	}), &fakeCacheRepository{
		restore: executioncache.RestoreResult{Hit: true, Exact: true, MatchedKey: "npm-abc", URL: server.URL},
	}, out)

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cache: miss")

	entries, readErr := os.ReadDir(root)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "a failed restore must not leave a partial tree behind")
	assert.Equal(t, executioncache.HitMiss, readState(t, statePath).Hit)
}

func TestRunCacheSave_SkipsOnExactHit(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	writeState(t, statePath, executioncache.State{Key: "npm-abc", Hit: executioncache.HitExact})

	repository := &fakeCacheRepository{}
	out := &bytes.Buffer{}

	require.NoError(t, runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{t.TempDir()},
	}), nil, statePath, cacheDefaultMaxSize, repository, out))

	assert.Contains(t, out.String(), "nothing to save")
	assert.Zero(t, repository.saveCalls, "an exact hit must not reach the control plane at all")
}

// TestRunCacheSave_UsesTheKeyRestoreResolved is the guard on the handshake. An install
// may rewrite the lockfile the key hashes, so recomputing it here could store the entry
// under a key no later run will search for.
func TestRunCacheSave_UsesTheKeyRestoreResolved(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "dep.txt"), []byte("installed"), 0o644))

	statePath := filepath.Join(t.TempDir(), "state.json")
	writeState(t, statePath, executioncache.State{
		Key: "npm-resolved-at-restore", Hit: executioncache.HitMiss, Paths: []string{root},
	})

	var uploaded bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploaded = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repository := &fakeCacheRepository{save: executioncache.SaveResult{URL: server.URL}}
	out := &bytes.Buffer{}

	require.NoError(t, runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-recomputed-differently",
		Paths: []string{root},
	}), []string{root}, statePath, cacheDefaultMaxSize, repository, out))

	assert.Equal(t, "npm-resolved-at-restore", repository.savedKey)
	assert.True(t, uploaded)
	assert.Positive(t, repository.savedSize, "the size must be known before the upload is granted")
	assert.Contains(t, out.String(), "cache: saved")
}

func TestRunCacheSave_DegradesQuietly(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "dep.txt"), []byte("installed"), 0o644))

	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer refusing.Close()

	cases := []struct {
		name       string
		repository executioncache.Repository
		wantOutput string
	}{
		{
			name:       "capability absent",
			repository: executioncache.Unsupported("caching is off"),
			wantOutput: "caching is off",
		},
		{
			name:       "method missing",
			repository: &fakeCacheRepository{saveErr: status.Error(codes.Unimplemented, "nope")},
			wantOutput: "cannot serve dependency caches",
		},
		{
			name:       "quota refused",
			repository: &fakeCacheRepository{saveErr: status.Error(codes.ResourceExhausted, "over quota")},
			wantOutput: "over quota",
		},
		{
			name:       "already stored by a concurrent execution",
			repository: &fakeCacheRepository{save: executioncache.SaveResult{AlreadyExists: true}},
			wantOutput: "already stored",
		},
		{
			name:       "upload rejected",
			repository: &fakeCacheRepository{save: executioncache.SaveResult{URL: refusing.URL}},
			wantOutput: "cache: not saving",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			err := runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
				Key:   "npm-abc",
				Paths: []string{root},
			}), []string{root}, "", cacheDefaultMaxSize, tc.repository, out)

			assert.NoError(t, err)
			assert.Contains(t, out.String(), tc.wantOutput)
		})
	}
}

func TestRunCacheSave_RefusesAnOversizedArchive(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "dep.txt"), bytes.Repeat([]byte("a"), 4096), 0o644))

	repository := &fakeCacheRepository{}
	out := &bytes.Buffer{}

	require.NoError(t, runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{root},
	}), []string{root}, "", 1, repository, out))

	assert.Contains(t, out.String(), "over the")
	assert.Zero(t, repository.saveCalls, "an oversized archive must not be granted an upload")
}

func TestRunCache_RejectsAMalformedPayload(t *testing.T) {
	out := &bytes.Buffer{}
	assert.Error(t, runCacheRestore(context.Background(), "not-base64!!", &fakeCacheRepository{}, out))
	assert.Error(t, runCacheRestore(context.Background(), "", &fakeCacheRepository{}, out))
	assert.Error(t, runCacheSave(context.Background(), "not-base64!!", nil, "", cacheDefaultMaxSize, &fakeCacheRepository{}, out))
}

// TestUnsupportedRepositoryNeverErrors pins the stand-in's behaviour, since it is what
// runs whenever caching is unavailable.
func TestUnsupportedRepositoryNeverErrors(t *testing.T) {
	repository := executioncache.Unsupported("because")

	restore, err := repository.Restore(context.Background(), executioncache.RestoreRequest{Key: "k"})
	assert.NoError(t, err)
	assert.False(t, restore.Hit)

	save, err := repository.Save(context.Background(), executioncache.SaveRequest{Key: "k"})
	assert.NoError(t, err)
	assert.True(t, save.AlreadyExists, "so the caller skips the upload without reporting a failure")

	assert.Equal(t, "because", executioncache.Reason(repository))
	assert.Empty(t, executioncache.Reason(&fakeCacheRepository{}))
}

func TestDegradedClassification(t *testing.T) {
	_, degraded := executioncache.Degraded(nil)
	assert.False(t, degraded)

	_, degraded = executioncache.Degraded(status.Error(codes.Unimplemented, "x"))
	assert.True(t, degraded)

	_, degraded = executioncache.Degraded(status.Error(codes.ResourceExhausted, "x"))
	assert.True(t, degraded)

	// An unexpected failure is not something a cache may swallow: it should surface so
	// that a genuine bug is visible rather than looking like a cold cache forever.
	_, degraded = executioncache.Degraded(errors.New("connection reset"))
	assert.False(t, degraded)

	_, degraded = executioncache.Degraded(status.Error(codes.Internal, "boom"))
	assert.False(t, degraded)
}

func readState(t *testing.T, path string) executioncache.State {
	t.Helper()
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	var state executioncache.State
	require.NoError(t, json.Unmarshal(contents, &state))
	return state
}

func writeState(t *testing.T, path string, state executioncache.State) {
	t.Helper()
	encoded, err := json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, encoded, 0o644))
}
