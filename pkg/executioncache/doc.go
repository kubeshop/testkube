// Package executioncache derives where a step's dependency cache lives in object
// storage, and picks which stored entry a restore should use.
//
// A step's cache block (StepCache in api/testworkflows/v1/step_types.go) restores
// directories from object storage before the step runs and saves them back once it
// passes, so dependency installs survive between executions. This package is the
// transport-free core of that: the object-key derivation (objectkey.go), the
// restore-key match policy (match.go), the payload and handshake shapes both sides
// share (args.go), and the repository interface plus its degrade-to-miss
// classification (repository.go).
//
// Entries are separated by namespace as well as by scope. A run for a pull request writes
// only into its own namespace and may read the base one; every trusted run - a push, a
// tag, a schedule, a manual run - reads and writes the base namespace. Without that, a
// pull request could store an entry that a later run for the default branch restored by
// way of a restoreKeys prefix, and for a build cache that is code that then runs. The
// author of a pull request controls what executes in its run, so the identity of the
// namespace has to be derived from what the trigger recorded, never from the request.
//
// Both halves live here, away from any transport, for one reason: the control plane in
// this repository and the commercial one must agree on them exactly. A disagreement
// would not fail loudly - it would produce entries that the other plane can never find,
// so every run would miss its own cache and quietly reinstall everything. The
// commercial plane must import this package rather than reimplement it.
//
// # The rest of the feature
//
// ProcessCacheRestore and ProcessCacheSave in
// pkg/testworkflows/testworkflowprocessor/operations_cache.go are registered in both
// presets. Restore sits after the content operations so the repository is checked out
// when the key is resolved; save sits after the step's work and before artifacts.
//
// cmd/testworkflow-toolkit/commands/cache.go is the pod-side half. It never exits
// non-zero: a cache is an optimization, so a miss, an unreachable Control Plane, a
// missing capability, a corrupt archive or a refused upload all leave the step to
// install from the network.
//
// Three constraints are easy to break and worth knowing before editing any of it:
//
//   - The specification travels base64-encoded in one argument. testworkflow-init
//     resolves every container argument with expressions.FinalizerFail and exits the
//     step on failure, so a key holding hash_files() over an absent lockfile would
//     kill the step rather than miss the cache. TestProcessCache_KeyTemplateStaysOpaque
//     guards this.
//   - The two stages hand the resolved key over through TK_CACHE_STATE on the shared
//     /testkube volume instead of each computing it. They are separate containers, and
//     an install may rewrite the very lockfile the key hashes (npm ci does), so a
//     recomputed key could store the entry where nothing later searches for it.
//   - Cached paths are mounted automatically. Each stage is its own container, and
//     containers share volumes but not their root filesystems, so a path outside every
//     volume is restored where the container running the install cannot see it.
//     "mount: false" on an uncovered path is refused at bundle time rather than
//     silently doing nothing.
//
// The save stage sets no condition, inheriting passed - deliberately unlike the
// artifacts stage, which is always: publishing a failed install under a content-hash
// key would poison every later run with no way for a user to invalidate it.
//
// hash() (pkg/expressions/stdlib.go) and hash_files() (pkg/expressions/libs/fs.go)
// exist for building keys. Prefer hash_files: hash(glob(...)) digests the matched
// paths, so it does not change when a file's contents do.
//
// Cache entries expire through STORAGE_CACHE_EXPIRATION; the minio package documents
// how that lifecycle rule is applied.
package executioncache
