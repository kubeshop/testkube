// Package localinstall powers the hidden `testkube install local` command.
//
// It prepares a laptop for a local Testkube trial.
// Today it only runs preflight checks; installing comes later.
//
// Checks (see Checker):
//   - Docker missing, unreachable or permission denied blocks the install.
//   - Missing kubectl, helm or kind only warns; we install them.
//   - Low Docker CPU, memory or disk only warns.
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
// Tracker reports each step to PostHog project "On Prem Trials"
// through https://t.testkube.io/capture/, keyed by the CLI machine ID
// plus a per-run install_session_id.
// Sends run in the background with a 2s timeout;
// call Wait before exit, since os.Exit skips defers.
// DO_NOT_TRACK or telemetryEnabled false sends nothing, hides the notice.
// Events: install_local_started, install_local_check,
// install_local_failed, install_local_checks_done.
package localinstall
