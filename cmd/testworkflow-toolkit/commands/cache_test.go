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
	"strings"
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
	// A cache is unpacked at "/", so the entry name and the declared cache path are both
	// derived from one POSIX form of the temp dir. On Linux that is the temp dir itself;
	// elsewhere the drive letter is dropped, which keeps the two consistent so the
	// allowlist is exercised rather than tripped by host path syntax.
	posix := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	archive := cacheTarball(t, strings.TrimPrefix(posix+"/restored.txt", "/"), "from-cache")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{posix},
		State: statePath,
	}), &fakeCacheRepository{
		restore: executioncache.RestoreResult{
			Hit: true, Exact: true, MatchedKey: "npm-abc", URL: server.URL, Size: int64(len(archive)),
		},
	}, out)

	require.NoError(t, err)
	assert.Contains(t, out.String(), "cache: hit")
	// The archive really was written, not merely reported: this also proves the
	// allowlist admits the paths the step declared.
	restored, readErr := os.ReadFile(filepath.Join(root, "restored.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "from-cache", string(restored))

	state := readState(t, statePath)
	assert.Equal(t, executioncache.HitExact, state.Hit)
	assert.Equal(t, "npm-abc", state.MatchedKey)
}

func TestRunCacheRestore_PartialHitIsRecordedSoSaveStillRuns(t *testing.T) {
	root := t.TempDir()
	posix := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	archive := cacheTarball(t, strings.TrimPrefix(posix+"/restored.txt", "/"), "older")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	require.NoError(t, runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:         "npm-new",
		RestoreKeys: []string{"npm-"},
		Paths:       []string{posix},
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
	// Container paths, as the code now resolves them: see TestRunCacheRestore_Hit.
	posix := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	require.NoError(t, os.WriteFile(filepath.Join(root, "stale.txt"), []byte("x"), 0o644))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("this is not a gzip stream"))
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{posix},
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

// requireContainerPaths skips a test that packs real directories through the cache save.
//
// A cache is packed rooted at "/" with container-absolute paths, which is what the
// walker and the allowlist agree on. A Windows temp directory is "C:\..." - not rooted
// at "/" and not a valid io/fs path - so the walk finds nothing there and the test would
// report a failure about the host rather than about the cache. This code only ever runs
// in Linux containers.
func requireContainerPaths(t *testing.T, dir string) {
	t.Helper()
	if filepath.VolumeName(dir) != "" {
		t.Skip("test packs container-absolute paths; a Windows drive path cannot be walked from \"/\"")
	}
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
	requireContainerPaths(t, root)
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
	requireContainerPaths(t, root)
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
	requireContainerPaths(t, root)
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

// TestRunCacheSave_RefusesAnEmptyArchive guards the consequence of entries being
// immutable: storing nothing under a key can never be undone.
//
// An empty archive would answer every later run with a hit that restores nothing, and no
// rerun could displace it, so the key would be poisoned for the lifetime of the entry.
// Nothing to pack means the paths were never created - the install wrote somewhere else,
// or the step had nothing to do - and either way there is nothing worth a key.
func TestRunCacheSave_RefusesAnEmptyArchive(t *testing.T) {
	// Empty rather than missing, so this is about there being no content rather than
	// about a path that cannot be read. Directories are not archived, so both pack
	// nothing - on every host, which is why this one runs where the tests above cannot.
	root := t.TempDir()

	repository := &fakeCacheRepository{}
	out := &bytes.Buffer{}

	require.NoError(t, runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{root},
	}), []string{root}, "", cacheDefaultMaxSize, repository, out))

	assert.Contains(t, out.String(), "nothing was found")
	assert.Zero(t, repository.saveCalls, "an empty archive must not be granted an upload")
}

// TestCachePackPatterns pins what the cached paths are turned into before the walker
// sees them. A bare path matches only the directory, which the walker skips, so without
// the "/**" form the archive comes out empty and the cache silently does nothing; without
// the bare form a cached path naming a single file is never packed.
func TestCachePackPatterns(t *testing.T) {
	assert.Equal(t,
		[]string{"/data/node_modules", "/data/node_modules/**", "/root/.m2", "/root/.m2/**"},
		cachePackPatterns([]string{"/data/node_modules", "/root/.m2"}))

	assert.Empty(t, cachePackPatterns(nil))
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

// stubCacheClient is the control-plane client the repository wraps.
type stubCacheClient struct {
	restore    executioncache.RestoreResult
	restoreErr error
	save       executioncache.SaveResult
	saveErr    error

	sawEnvironment string
	sawExecution   string
	sawScope       executioncache.Scope
}

func (s *stubCacheClient) GetExecutionCachePresignedURL(_ context.Context, environmentId, executionId string, req executioncache.RestoreRequest) (executioncache.RestoreResult, error) {
	s.sawEnvironment, s.sawExecution, s.sawScope = environmentId, executionId, req.Scope
	return s.restore, s.restoreErr
}

func (s *stubCacheClient) SaveExecutionCacheGetPresignedURL(_ context.Context, environmentId, executionId string, req executioncache.SaveRequest) (executioncache.SaveResult, error) {
	s.sawEnvironment, s.sawExecution, s.sawScope = environmentId, executionId, req.Scope
	return s.save, s.saveErr
}

// TestControlPlaneCacheRepository checks the wiring between the commands and the
// control-plane client: the pod's own identity is carried, not taken per call, and the
// scope travels through untouched.
func TestControlPlaneCacheRepository(t *testing.T) {
	stub := &stubCacheClient{restore: executioncache.RestoreResult{Hit: true, Exact: true, URL: "https://storage/x"}}
	repository := controlPlaneCacheRepository{client: stub, environmentID: "env-1", executionID: "exec-1"}

	got, err := repository.Restore(context.Background(), executioncache.RestoreRequest{
		Key: "npm-abc", Scope: executioncache.ScopeEnvironment,
	})
	require.NoError(t, err)
	assert.True(t, got.Hit)
	assert.Equal(t, "env-1", stub.sawEnvironment)
	assert.Equal(t, "exec-1", stub.sawExecution)
	assert.Equal(t, executioncache.ScopeEnvironment, stub.sawScope)

	stub.save = executioncache.SaveResult{URL: "https://storage/put"}
	saved, err := repository.Save(context.Background(), executioncache.SaveRequest{
		Key: "npm-abc", Scope: executioncache.ScopeWorkflow, Size: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://storage/put", saved.URL)
	assert.Equal(t, executioncache.ScopeWorkflow, stub.sawScope)
}

// TestControlPlaneCacheRepositoryPropagatesForDegrading: the repository does not swallow
// errors itself. It hands them up so that runCacheRestore/runCacheSave classify them -
// Unimplemented and a refusal become a miss, anything else stays visible.
func TestControlPlaneCacheRepositoryPropagatesForDegrading(t *testing.T) {
	stub := &stubCacheClient{restoreErr: status.Error(codes.Unimplemented, "no such method")}
	repository := controlPlaneCacheRepository{client: stub, environmentID: "env-1", executionID: "exec-1"}

	_, err := repository.Restore(context.Background(), executioncache.RestoreRequest{Key: "npm-abc"})
	require.Error(t, err)
	assert.True(t, executioncache.IsUnsupported(err))

	// And the command layer turns exactly that into a miss rather than a failure.
	out := &bytes.Buffer{}
	assert.NoError(t, runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key: "npm-abc", Paths: []string{t.TempDir()},
	}), repository, out))
	assert.Contains(t, out.String(), "cannot serve dependency caches")
}

// TestRunCacheRestore_RejectsAnArchiveReachingOutsideTheDeclaredPaths is the command-level
// half of the allowlist.
//
// The archive is written by whoever populated the entry, and under an environment-scoped
// cache that is another workflow. An entry outside the paths this step declared must not
// be written - and because a filtered tree is not a tree the step asked for, the whole
// restore is abandoned: the declared paths are cleared and the step reports a miss so it
// rebuilds from scratch.
func TestRunCacheRestore_RejectsAnArchiveReachingOutsideTheDeclaredPaths(t *testing.T) {
	root := t.TempDir()
	// Container paths, as the code now resolves them: see TestRunCacheRestore_Hit.
	posix := filepath.ToSlash(root[len(filepath.VolumeName(root)):])
	declaredPosix := posix + "/deps"
	declared := filepath.Join(root, "deps")
	require.NoError(t, os.MkdirAll(declared, 0o755))
	// Something the previous step left behind, which the failed restore must clear.
	require.NoError(t, os.WriteFile(filepath.Join(declared, "stale.txt"), []byte("x"), 0o644))

	elsewhere := filepath.Join(root, "elsewhere")
	require.NoError(t, os.MkdirAll(elsewhere, 0o755))

	// Names are relative to "/", which is where a cache archive is unpacked.
	outside := strings.TrimPrefix(filepath.ToSlash(filepath.Join(elsewhere, "planted.txt")), "/")
	archive := cacheTarball(t, outside, "planted-by-another-workflow")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	out := &bytes.Buffer{}

	err := runCacheRestore(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{declaredPosix},
		State: statePath,
	}), &fakeCacheRepository{
		restore: executioncache.RestoreResult{Hit: true, Exact: true, MatchedKey: "npm-abc", URL: server.URL},
	}, out)

	assert.NoError(t, err, "a hostile archive is a miss, not a failed step")
	assert.Contains(t, out.String(), "cache: miss")

	_, statErr := os.Stat(filepath.Join(elsewhere, "planted.txt"))
	assert.True(t, os.IsNotExist(statErr), "the archive wrote outside the declared paths")

	entries, readErr := os.ReadDir(declared)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "the declared paths must be cleared after a refused restore")

	assert.Equal(t, executioncache.HitMiss, readState(t, statePath).Hit)
}

// TestRunCacheSave_SendsTheConditionalHeaders is what makes an entry immutable.
//
// The control plane signs the condition into the upload, so the agent has to send the
// headers verbatim: an upload without them is rejected rather than silently becoming a
// plain overwrite.
func TestRunCacheSave_SendsTheConditionalHeaders(t *testing.T) {
	root := t.TempDir()
	requireContainerPaths(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "dep.txt"), []byte("installed"), 0o644))

	var seen http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	out := &bytes.Buffer{}
	require.NoError(t, runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
		Key:   "npm-abc",
		Paths: []string{root},
	}), []string{root}, "", cacheDefaultMaxSize, &fakeCacheRepository{
		save: executioncache.SaveResult{
			URL:     server.URL,
			Headers: map[string]string{"If-None-Match": "*"},
		},
	}, out))

	assert.Equal(t, "*", seen.Get("If-None-Match"), "the signed condition must reach the store")
	assert.Contains(t, out.String(), "cache: saved")
}

// TestRunCacheSave_ARefusedUploadIsNotAFailure covers losing the race the condition
// creates: exactly one of two executions saving a key can succeed, and the loser is
// refused. That is a normal outcome, because the winner stored an equivalent tree under
// the same content-derived key.
func TestRunCacheSave_ARefusedUploadIsNotAFailure(t *testing.T) {
	root := t.TempDir()
	requireContainerPaths(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "dep.txt"), []byte("installed"), 0o644))

	for _, refusal := range []int{http.StatusPreconditionFailed, http.StatusConflict} {
		t.Run(http.StatusText(refusal), func(t *testing.T) {
			var attempts int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.WriteHeader(refusal)
			}))
			defer server.Close()

			out := &bytes.Buffer{}
			err := runCacheSave(context.Background(), encodeCacheArgs(t, executioncache.Args{
				Key:   "npm-abc",
				Paths: []string{root},
			}), []string{root}, "", cacheDefaultMaxSize, &fakeCacheRepository{
				save: executioncache.SaveResult{
					URL:     server.URL,
					Headers: map[string]string{"If-None-Match": "*"},
				},
			}, out)

			assert.NoError(t, err)
			assert.Contains(t, out.String(), "stored by another execution first")
			assert.Equal(t, 1, attempts, "a refused condition will refuse again, so it must not be retried")
		})
	}
}

func TestUploadRefused(t *testing.T) {
	assert.True(t, executioncache.UploadRefused(http.StatusPreconditionFailed))
	assert.True(t, executioncache.UploadRefused(http.StatusConflict))
	assert.False(t, executioncache.UploadRefused(http.StatusOK))
	assert.False(t, executioncache.UploadRefused(http.StatusForbidden))
	assert.False(t, executioncache.UploadRefused(http.StatusInternalServerError))
}
