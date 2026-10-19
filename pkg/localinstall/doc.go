// Package localinstall powers the `testkube install local` command.
//
// It prepares a laptop for a local Testkube trial.
// It checks the machine, prepares tools, creates the cluster
// and installs Testkube.
//
// The license key comes first (see LicenseStep):
// --license or TESTKUBE_LICENSE gets one try, a masked prompt gets three.
// An unreachable license.testkube.io stops the install,
// because the installed control plane would fail the same way.
// An expired key stops at once; retyping it can't help.
//
// Checks (see Checker):
//   - Docker missing, unreachable or permission denied blocks the install.
//   - So does Podman's docker stand-in; kind always uses Docker.
//   - For those two (not WSL), install also offers init demo.
//   - First runs with a kube context ask (CurrentKubeContext).
//   - Missing kubectl only warns; we install it.
//   - helm and kind are always our pinned copies, never the user's.
//   - Low Docker CPU, memory or disk only warns.
//   - Low inotify limits only warn, when Docker shares this kernel.
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
// A new cluster pulls Docker Hub images through mirror.gcr.io
// when it answers, falling back to Docker Hub, to avoid
// Docker Hub's per-network pull limit.
//
// Testkube (see Installer) comes from pinned charts and
// embedded values, installed by our helm (see Helm).
// Passwords the kept data needs live in secrets.json,
// so a rebuilt cluster can still open its old data.
// A failed install names the root stuck service (see Stuck);
// helm stops early when waiting can't help, like a missing image.
// Meanwhile OnProgress reports each service's state and downloads.
// It also saves a private report for support (see saveReport),
// built from picked fields only, with known secrets masked.
// CLIKey logs in like the dashboard and mints a CLI API key,
// so the testkube CLI can use the install without a browser.
// ReadStatus reports the trial's state without starting anything.
// Uninstall (see Cluster.Remove, RemoveFiles, DeleteData) keeps data
// unless asked, and never removes a cluster it did not make.
//
// Step tracking lives in pkg/telemetry, see InstallTracker.
package localinstall
