// Package localinstall powers the hidden `testkube install local` command.
//
// It prepares a laptop for a local Testkube trial.
// Today it checks the machine, prepares tools and creates the cluster;
// the Testkube install comes later.
//
// The license key comes first (see LicenseStep):
// --license or TESTKUBE_LICENSE gets one try, a masked prompt gets three.
// An unreachable license.testkube.io stops the install,
// because the installed control plane would fail the same way.
//
// Checks (see Checker):
//   - Docker missing, unreachable or permission denied blocks the install.
//   - Missing kubectl only warns; we install it.
//   - helm and kind are always our pinned copies, never the user's.
//   - Low Docker CPU, memory or disk only warns.
//   - Unreachable download sites only warn; Docker may use its own proxy.
//
// Resources come from one `docker info --format '{{json .}}'` call,
// so Docker Desktop's VM limits apply, not the host's.
// docker info gives up after 10 seconds.
// Free disk is read at Docker's data dir;
// skipped when that dir isn't on this host (Docker Desktop).
//
// Checker reaches Docker and the machine through private interfaces;
// NewChecker wires the real ones, tests pass fakes.
//
// Missing tools (see ToolInstaller) are downloaded at pinned versions,
// checksum-verified, into ~/.testkube/bin. That folder is appended
// to the CLI's own PATH, so the user's versions still win.
//
// The cluster (see Cluster) is a kind cluster named "testkube":
// its kubeconfig is ~/.testkube/kubeconfig, never ~/.kube/config;
// five browser ports map to 127.0.0.1 only, next free when busy;
// ~/.testkube/data holds its storage, so data outlives the cluster.
// Re-runs reuse it only when it carries the random
// mark saved in ~/.testkube/cluster-id at creation,
// read from its mounts before starting a stopped node.
//
// Step tracking lives in pkg/telemetry, see InstallTracker.
package localinstall
