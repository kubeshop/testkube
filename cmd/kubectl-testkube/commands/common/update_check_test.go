package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cliruntime"
)

// withStubFetcher swaps the package-private fetchLatestVersion seam used by
// MaybeNotifyNewerRelease and CheckComponentsStatus. The stub captures whether
// it was called so tests can assert the cache short-circuit path.
type fetcherStub struct {
	value  string
	err    error
	called int
}

func (s *fetcherStub) fetch(_ context.Context) (string, error) {
	s.called++
	return s.value, s.err
}

func withFetcher(t *testing.T, stub *fetcherStub) {
	t.Helper()
	original := fetchLatestVersion
	fetchLatestVersion = stub.fetch
	t.Cleanup(func() { fetchLatestVersion = original })
}

// withVersion temporarily overrides the build-injected CLI version for tests
// that exercise the semver-parse branches of MaybeNotifyNewerRelease.
func withVersion(t *testing.T, version string) {
	t.Helper()
	original := Version
	Version = version
	t.Cleanup(func() { Version = original })
}

// withRunContext swaps the cliContext detection seam so each test can simulate
// a specific runtime (local, CI system, container) regardless of the host
// environment. Also clears the env opt-out so tests start from a known state.
func withRunContext(t *testing.T, ctx string) {
	t.Helper()
	t.Setenv("TESTKUBE_DISABLE_UPDATE_CHECK", "")
	original := detectCliRunContext
	detectCliRunContext = func() string { return ctx }
	t.Cleanup(func() { detectCliRunContext = original })
}

// withLocalRunContext is the common case for the per-command hint tests: the
// CLI is on a developer machine. Use withRunContext to simulate CI/Docker.
func withLocalRunContext(t *testing.T) {
	t.Helper()
	withRunContext(t, cliruntime.CliRunContextLocal)
}

// newCmdWithOutputFlag returns a minimal cobra.Command with an --output flag
// preset to the requested value. It is sufficient for tests that only need
// the output gate to behave correctly.
func newCmdWithOutputFlag(name, output string) *cobra.Command {
	cmd := &cobra.Command{Use: name}
	cmd.Flags().StringP("output", "o", output, "")
	return cmd
}

func TestMaybeNotifyNewerRelease_SkipsVersionCommand(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{}

	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false for version command")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called for version command, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_SkipsEnvOptOut(t *testing.T) {
	withLocalRunContext(t)
	t.Setenv("TESTKUBE_DISABLE_UPDATE_CHECK", "1")
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{}

	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false when env opt-out is set")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called when opt-out is set, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_SkipsNonPrettyOutput(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "json")
	cfg := &config.Data{}

	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false for non-pretty output")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called for non-pretty output, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_SkipsCIContext(t *testing.T) {
	withRunContext(t, "github-actions")
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{}

	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false in CI context")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called in CI context, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_SkipsUnparseableClientVersion(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "dev")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{}

	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false when client version is not semver")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called for unparseable version, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_UsesFreshCache(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{
		LastUpdateCheckAt:  time.Now().Add(-1 * time.Hour),
		LatestKnownVersion: "2.1.135",
	}

	dirty := MaybeNotifyNewerRelease(cmd, cfg)
	if dirty {
		t.Fatal("expected dirty=false when reading from fresh cache")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called when cache is fresh, got %d calls", stub.called)
	}
}

func TestMaybeNotifyNewerRelease_RefreshesStaleCache(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{
		LastUpdateCheckAt:  time.Now().Add(-48 * time.Hour),
		LatestKnownVersion: "2.1.135",
	}

	dirty := MaybeNotifyNewerRelease(cmd, cfg)
	if !dirty {
		t.Fatal("expected dirty=true after refreshing stale cache")
	}
	if stub.called != 1 {
		t.Fatalf("fetcher should be called exactly once, got %d", stub.called)
	}
	if cfg.LatestKnownVersion != "2.1.140" {
		t.Fatalf("expected LatestKnownVersion=2.1.140, got %q", cfg.LatestKnownVersion)
	}
}

func TestMaybeNotifyNewerRelease_NoOpWhenUpToDate(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)
	withVersion(t, "2.1.140")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{
		LastUpdateCheckAt:  time.Now().Add(-1 * time.Hour),
		LatestKnownVersion: "2.1.140",
	}

	dirty := MaybeNotifyNewerRelease(cmd, cfg)
	if dirty {
		t.Fatal("expected dirty=false when versions match and cache is fresh")
	}
}

func TestMaybeNotifyNewerRelease_HTTPFailureIsSilent(t *testing.T) {
	withLocalRunContext(t)
	stub := &fetcherStub{err: errors.New("network down")}
	withFetcher(t, stub)
	withVersion(t, "2.1.130")

	cmd := newCmdWithOutputFlag("get", "pretty")
	cfg := &config.Data{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HTTP failure caused panic: %v", r)
		}
	}()
	if dirty := MaybeNotifyNewerRelease(cmd, cfg); dirty {
		t.Fatal("expected dirty=false on fetch error")
	}
}

func TestCheckComponentsStatus_SkipsNonPrettyOutput(t *testing.T) {
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "yaml")
	cfg := &config.Data{}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", "2.1.125"); dirty {
		t.Fatal("expected dirty=false for non-pretty output")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called for non-pretty output, got %d calls", stub.called)
	}
}

func TestCheckComponentsStatus_SkipsEnvOptOut(t *testing.T) {
	t.Setenv("TESTKUBE_DISABLE_UPDATE_CHECK", "1")
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", "2.1.125"); dirty {
		t.Fatal("expected dirty=false when env opt-out is set")
	}
	if stub.called != 0 {
		t.Fatalf("fetcher should not be called when opt-out is set, got %d calls", stub.called)
	}
}

func TestCheckComponentsStatus_RunsInCIContext(t *testing.T) {
	// Unlike MaybeNotifyNewerRelease this entry point is explicit and must
	// run even when cliContext is something other than the local sentinel.
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("TESTKUBE_DISABLE_UPDATE_CHECK", "")
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", "2.1.125"); !dirty {
		t.Fatal("expected dirty=true after successful fetch")
	}
	if stub.called != 1 {
		t.Fatalf("fetcher should be called even in CI, got %d", stub.called)
	}
}

func TestCheckComponentsStatus_CachesFreshValue(t *testing.T) {
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", "2.1.125"); !dirty {
		t.Fatal("expected dirty=true so the caller can persist the new cache")
	}
	if cfg.LatestKnownVersion != "2.1.140" {
		t.Fatalf("expected LatestKnownVersion=2.1.140, got %q", cfg.LatestKnownVersion)
	}
	if cfg.LastUpdateCheckAt.IsZero() {
		t.Fatal("expected LastUpdateCheckAt to be set")
	}
}

func TestCheckComponentsStatus_FetchFailureIsRecoverable(t *testing.T) {
	stub := &fetcherStub{err: errors.New("network down")}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{ContextType: config.ContextTypeKubeconfig}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("fetch failure caused panic: %v", r)
		}
	}()
	dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", "2.1.125")
	if dirty {
		t.Fatal("expected dirty=false when fetch fails")
	}
	if stub.called != 1 {
		t.Fatalf("fetcher should be invoked exactly once, got %d", stub.called)
	}
}

func TestCheckComponentsStatus_DisconnectedServerSkipsServerBlock(t *testing.T) {
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{ContextType: config.ContextTypeKubeconfig}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.130", ""); !dirty {
		t.Fatal("expected dirty=true so the CLI block still updates the cache")
	}
}

func TestCheckComponentsStatus_CloudContextSuppressesServerHint(t *testing.T) {
	stub := &fetcherStub{value: "2.1.140"}
	withFetcher(t, stub)

	cmd := newCmdWithOutputFlag("version", "pretty")
	cfg := &config.Data{ContextType: config.ContextTypeCloud}

	if dirty := CheckComponentsStatus(cmd, cfg, "2.1.140", "2.1.125"); !dirty {
		t.Fatal("expected dirty=true after successful fetch")
	}
}
