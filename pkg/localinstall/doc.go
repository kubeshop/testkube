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
// Free disk is read at Docker's data dir;
// Docker Desktop hides that dir in its VM: skipped.
//
// Checker reaches Docker and the machine through private interfaces;
// NewChecker wires the real ones, tests pass fakes.
package localinstall
