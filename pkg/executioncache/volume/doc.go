// Package volume stores a cache archive on a shared ReadWriteMany volume instead of in
// the object store, leaving the object store holding a pointer to it.
//
// The split is deliberate. Everything that decides *who may read or write an entry*
// stays in the object store, where the control plane mediates it from the stored
// execution; the volume only holds bytes, under a path kubelet confines each pod to. An
// execution can therefore never publish an entry that another execution's scope would
// restore, however it behaves inside its pod - which is the property the presigned
// layout provides and the one a plain shared mount would lose.
//
// Two consequences worth stating, because they are what make the arrangement cheap:
//
// The race is still resolved by the conditional write on the pointer object, so an
// entry is immutable for exactly the reason it already was. Nothing here has to
// reproduce that against a filesystem.
//
// A key never becomes a path segment. The object layout budgets a key against the
// object store's 1024-byte limit, which is three times what a POSIX filename may hold;
// naming files on the volume after keys would have broken for long keys only, which is
// the worst way to find out.
//
// Like the rest of executioncache, this format is shared with the commercial control
// plane rather than restated there. A disagreement about it would not fail loudly - it
// would produce entries the other side reads as archives and cannot unpack.
//
// # Overview
//
// With TESTKUBE_STEP_CACHE_VOLUME_CLAIM set, a cache object holds a few hundred bytes
// of pointer naming a directory on a ReadWriteMany volume instead of the archive
// itself. An entry is a directory mirroring the filesystem from / (under <entry>/root),
// so a restore is a copy rather than an unpack and nothing is gzipped.
//
// # Format and authorization
//
// The object store is still the index. Object name, resolveCacheScope, the namespaces
// and the If-None-Match conditional write are all unchanged, so a pull request still
// cannot publish under a key a trusted run restores - it still cannot write that
// object - and the race on one key is still resolved where it always was. Nothing about
// authorization moves onto the volume, and a change that moves it there would undo the
// whole point.
//
// The format is its own fallback, but only for entries this installation can still
// address. A body that does not begin with Magic is the archive, so an object
// holding one restores through the path it always did - which is what lets the
// object-store save branch, and a cluster without a volume, keep working unchanged. It
// does not reach entries written before keys were partitioned by volume: those live
// under the unscoped key and a volume-enabled runner now asks only for
// <volume id>/<key>, so they are unreachable and go cold until they expire. That is a one-time
// cost on upgrade, bounded by STORAGE_CACHE_EXPIRATION (default one day). A migration
// lookup for the unscoped key was considered and not taken: it would add a second round
// trip to every miss, permanently, and would have to refuse an unscoped pointer - which
// may name an entry on a volume this runner cannot read - to avoid reintroducing the
// collision partitioning exists to prevent.
//
// SaveRequest.Size is the object's size, so on this path it is the pointer's. The
// contract is what the bucket is about to receive, which a quota can refuse before the
// transfer; the tree is on a volume the Control Plane neither provisions nor can
// measure. Sending the tree's size would let a quota refuse a save that stores a few
// hundred bytes. The tree is bounded instead by the agent's own size limit and
// reclaimed by the sweep. This is also why Commit runs before Save here, unlike the
// archive path: the pointer does not exist until the entry has a published name, and a
// committed entry that no pointer names is discarded on every path that does not
// publish.
//
// # Mounts and confinement
//
// The mounts are the confinement, and they are kubelet's. Both cache stages are
// SetPure(true), so action.Group merges them into the step's own container and
// CreateContainer unions the volume mounts - the step's own command therefore holds
// whatever the cache stages mount. The read side is the whole volume read-only; the
// write side is a subPath resolving to inbox/<root execution id> - the execution, not
// the pod, so that a parallel or service worker writes into the inbox the agent already
// made for its parent (nothing inside a pod can reach the volume root to make one of
// its own). Dropping either would let any step rewrite any entry, so
// processor_cache_test.go pins both.
//
// Everything on the volume is readable by every execution in the cluster. That is the
// accepted cost of sharing one, stated in the Helm value's own documentation.
//
// And the step's own command holds the writable inbox mount, which is a deliberate
// decision rather than an oversight. Both cache stages are pure, so action.Group merges
// them into the step's container: the subPath confines where a pod may write, to its
// own execution's inbox, but not what it writes there. A workflow can therefore write
// past CopyLimits - which bound what the cache copies, not what the step does - and can
// leave directories only its own user may enter, which the agent can never reclaim,
// being neither root nor the owner with fsGroup inoperative on NFS. The second is
// permanent and outlives the execution. Do not treat the limits or SharedDirMode as a
// security boundary; they keep the cache's own writes well-behaved and nothing more.
// The boundary is that the volume is only enabled where every workflow in the
// environment is trusted with it, which both charts' values say at length. Making
// ProcessCacheSave non-pure would move the mount into a toolkit container of its own
// and close this, at one extra init container per cached step - that is the change to
// make if the trust assumption ever stops holding.
//
// # Volume identity
//
// Cache keys are partitioned by the volume they are stored on. A key is shared by every
// runner in an environment - cacheScope carries the environment, workflow, scope and
// namespace, and no runner or volume - while an entry on a volume is reachable only
// from that volume, and spec.target decides which runner runs an execution.
// Unpartitioned, whichever runner saved first owned the key for its whole lifetime:
// every runner on another volume got an exact hit it could not follow and could not
// replace, because the object is immutable. EnsureID writes a stable id to
// <volume>/.volume-id, the agent carries it to the pod in StepCacheVolumeConfig.ID and
// TK_CACHE_VOLUME_ID (a pod cannot read the volume root on the save side - its only
// writable mount is a subPath), and ScopedKey prefixes the key and every restore
// key with it. A prefix, not a suffix, so prefix matching cannot stray onto another
// volume's entry. The prefix is spent out of MaxKeyBytes, so the key is validated after
// scoping (validateScopedCacheKey) - checking the author's key and then adding bytes to
// it let a key just inside the limit pass in the pod and be refused by the API on every
// restore and save. The id belongs to the volume rather than the runner, so runners
// that do share a volume go on sharing its entries. O_EXCL publishes the name before
// the contents, so an agent killed in between leaves an empty .volume-id that would
// otherwise turn the cache off for the whole installation for good - every later
// startup waiting out the attempts and refusing the volume. One old enough not to be
// anybody's open write is superseded, never repaired and never unlinked: the agent
// moves to the next generation of the name (.volume-id.2, then .volume-id.3, bounded by
// idGenerations), and a generation that exists always wins over the one before it, so
// an abandoned name written long afterwards cannot take the volume back off the agents
// using its successor. The agent that was declared abandoned must not keep its own
// identity either, and that is closed on both sides, because one side cannot do it:
// after writing, a creator looks for a successor and adopts it (errIDSuperseded); and a
// new generation decides what it holds only once its O_EXCL create is won, inheriting
// whatever the generation it supersedes has gained by then (inheritedID). A successor
// made before that look is found by it, one made after it inherits the identity just
// written, and there is no ordering in which the two disagree. That look covers every
// path out of ensureID, not just the one that writes: an agent waiting out an empty
// generation holds no opinion for a second at a time, long enough for another to
// supersede it and the stalled creator to fill the old name, so a waiter can be handed
// an identity that was superseded while it waited. Note this needs no five-minute stall
// to reach: an mtime set by a skewed creating node can make an identity look abandoned
// the moment it is made, so idStaleAfter is a margin and not a guarantee. Nothing in
// identity.go unlinks anything, which is the whole point: every name is settled by an
// atomic O_EXCL create and no name is ever reused, so there is no instant at which one
// agent can remove what another has just created. Repairing in place instead needs
// exclusion, and a lock on a shared filesystem cannot be reclaimed safely - reclaiming
// it means unlinking a path on the strength of a stat taken earlier, which is the very
// race the lock exists to prevent, so a crash mid-repair would either wedge the volume
// for good or let two agents write different identities into it. The mode is set before
// the contents for the same family of reasons - a death in between would otherwise
// leave a non-empty identity at the creator's umask that no other agent can read and
// that the repair deliberately will not touch, a file with contents being somebody's
// identity rather than a half-made one.
//
// # Inboxes and leases
//
// The agent makes each execution's inbox before its pod starts (prepareStepCacheInbox
// in kubernetesworker/worker.go, called at both deploy sites), keyed on the execution's
// root id so that one inbox serves an execution and everything it spawns - a parallel
// or service worker builds its pods from inside a pod, where nothing can reach the
// volume root, so it cannot make an inbox of its own. For the same reason it is made
// for every execution the volume is enabled for, and deliberately not gated on this
// bundle mounting the claim: a workflow may cache only inside a nested worker, whose
// spec is bundled later, so the root bundle proves nothing. The empty directories that
// costs are the sweep's to reclaim - an inbox holding no entries goes as soon as its
// lease is stale, well before the retention a real entry gets, since retention exists
// to outlive a pointer and an empty inbox has none.
//
// kubelet would create the subPath directory itself - it creates a missing one and its
// parents - but owned by root, with the mode of the volume root and no regard for
// runAsUser; fsGroup is not applied to a multi-writer volume, since the in-tree NFS
// plugin never reads it. A step running as another user would then be unable to write
// to its own inbox and would fall back to the object store on every save, so the cache
// would look configured and never once be used. kubelet takes an existing directory as
// it is, so the agent makes it first, mode 0777 because it cannot know which user the
// step runs as. Doing so also keeps the shared inbox parent out of kubelet's hands:
// creating a subPath is not tolerant of a concurrent creation before Kubernetes v1.37,
// so two executions starting together on one node while that parent was still missing
// could fail to start. It is best effort - a failure logs and continues, because
// kubelet still creates the directory and the pod-side probe still falls back. Note
// this needs the agent's claim and the execution namespace's claim to share backing
// storage, which the chart documents.
//
// A running execution holds its inbox through a lease. The inbox is made before the pod
// starts and its mtime only moves when something is staged inside it, so an execution
// that has cached nothing yet, or runs one long step, looks expired to the sweep
// however alive it is - and unlinking it leaves the pod writing through its subPath to
// an inode nothing can reach, then publishing a pointer naming a path that is gone.
// TouchLease refreshes <inbox>/.lease, Sweeper.LeaseTTL skips an inbox whose
// lease is fresh - but only when its age is non-negative, within a five minute skew
// window, because the step's own command holds the inbox mount and could otherwise date
// .lease a century ahead and pin that directory for good; the inbox's own mtime is
// bounded the same way, and refreshStepCacheLeases (cmd/api-server/stepcachelease.go)
// renews them every 5 minutes against a 30 minute TTL, inside the sweeper task so the
// first renewal precedes the first sweep. The live set is read back from the cluster
// (worker.List with Finished: false) rather than tracked in memory: a registration that
// is never cleared would pin an inbox for good, where a listing simply stops returning
// what has stopped running. Time-based for the same reason - a lease nobody refreshes
// goes stale and the sweep carries on. Nothing is swept until that first listing
// succeeds (awaitStepCacheLeases): a leader that has just taken over holds no leases,
// so a failed first pass would make every live inbox look abandoned. A later pass may
// fail safely, falling back on the leases the previous one wrote, which is why the
// interval is well under the TTL. A lease is how one agent tells another - an api and a
// runner sharing a volume hold separate leader elections and know only their own
// executions - and within one agent it is an indirection that can fail, because the
// step holds its inbox's mount and can create .lease itself, as its own user and umask.
// A lease the agent cannot write is replaced rather than given up on (unlinking needs
// the write bit on the 0777 inbox, not ownership of the file), and Sweeper.Protected is
// the backstop: the live set from the last successful listing is held in memory
// (liveInboxes) and consulted before the lease and the mtime, so what this agent knows
// directly about its own executions never depends on reading it back off the volume.
// Without it a renewal that keeps failing leaves a running execution's inbox looking
// abandoned, and the empty-inbox branch takes it 30 minutes later while its pod is
// still writing through the subPath.
//
// Every directory written onto the volume is chmod'ed to SharedDirMode (0777). A step
// container creates them under its own umask, as whatever user the workflow chose,
// while the agent sweeps them as another; a directory's write bit is what permits
// unlinking what is inside it, so one left at 0755 is a subtree the agent can never
// reclaim. Mkdir's mode argument is not enough - the usual 022 turns 0777 into 0755
// before it reaches the filesystem - so mkdirShared sets it explicitly, and
// mkdirAllShared covers the prefix above a declared path's root, which the walk never
// visits. The sweep also reports a removal it could not make and continues, rather than
// returning at the first: one stuck inbox must not stop it reclaiming everything after
// it.
//
// # Sweep
//
// The sweep is the agent's, not the control plane's. The control plane is not
// necessarily in the cluster (cmd/api-server/services/executionworker.go points pods at
// TestkubeProURL), so it cannot reach the volume. Sweeper runs as a leader-gated
// task in cmd/api-server/main.go and expires whole inboxes by mtime. Retention shorter
// than a pointer's lifetime is raised to it rather than warned about and used: an entry
// outliving its pointer only wastes space, where a pointer outliving its entry turns
// every hit on that key into a miss nothing can explain, and the key is immutable, so
// no later run can replace it until the object expires. That lifetime is
// PointerLifetime, and it is the earlier of both lifecycle rules, not just
// STORAGE_CACHE_EXPIRATION - the bucket-wide STORAGE_EXPIRATION rule is deliberately
// unfiltered and so expires cache objects too, so with the cache rule disabled a
// pointer still lives the bucket-wide span. With neither rule set, nothing expires a
// pointer, no retention can outlast one, and the agent says so at startup instead of
// raising retention without bound. Both charts mount the claim on their own Deployment,
// because a runner-only installation has no api Deployment and the sweep would
// otherwise read its missing inbox directory as an empty volume and let the claim fill.
//
// The volume is enabled, and swept, in every mode. commons.StepCacheVolumeEnabled
// answers on the claim alone, and both the mounts and the sweep ask it. The disk is the
// agent's own and is written to in every mode, so it has to be bounded in every mode; a
// shared volume filling without limit takes the cluster's storage with it.
//
// Only one half of the arrangement can be confirmed, and it is not the disk. An entry
// is reachable only through the object naming it, so a sweep is safe exactly when that
// object also goes away. This process installs that rule only where it owns the
// bucket - standalone mode, through SetExpirationPolicies, whose failure is logged while
// startup continues because reading the existing lifecycle needs a permission an
// upgrade may not have granted. An agent or runner attached to a Control Plane owns no
// bucket: it never calls SetExpirationPolicies, cannot read the lifecycle, and must be
// told how long a cache object lives there - stepCacheVolume.objectExpirationDays on
// the runner chart, storage.cacheExpiration on the api chart, both reaching
// STORAGE_CACHE_EXPIRATION. Retention is still raised to that lifetime
// (PointerLifetime). Where nothing is known to expire a pointer, the sweep runs
// anyway on the configured retention and the risk is stated once at startup: an entry
// swept before its pointer expires leaves a key that restores nothing until the object
// goes, and the key is immutable, so nothing repairs it meanwhile.
//
// The sweep rests on an assumption it cannot verify, and this is the known residual
// risk of the whole arrangement. Deleting an entry is safe only once the object naming
// it has stopped being served, and the agent has no way to know that: a day-based
// lifecycle rule is not a deadline, the store rounds it up to a UTC midnight and then
// removes asynchronously with no bound promised on the lag. stepCacheLifecycleMargin
// (48h, on top of the pointer's configured lifetime and the publication grace) is sized
// for the rounding and an ordinary delay, so it narrows the window - but a store slow
// enough still leaves a pointer answering with a hit to an entry that is gone, under a
// key no run can replace until the object finally expires. Do not read the margin as a
// guarantee; widening it buys probability, not certainty. Closing it needs a positive
// signal, and the agent cannot obtain one - attached to a Control Plane it has no
// access to the bucket its pointers go to, which is the same wall the expiration
// confirmation runs into above. The fix belongs to the Control Plane: deleting or
// replacing a pointer when a restore reports its entry missing, or reporting removals
// the agent can act on. Both are proto and testkube-cloud-api changes, so this package
// documents the risk rather than carrying a half of them.
//
// # Entries
//
// The entry limit is the volume's own, and a refusal on it falls back. The volume
// counts every directory where the archive's walker emits only files and links, so a
// directory holding two files is three entries here and two there. The archive path
// enforces its own count before asking for a grant, which is what keeps an entry no
// restore would accept out of the store - so falling back cannot publish one, and
// settling here would drop a cache the archive had room for. The size limit falls back
// for the same reason, measuring the tree where the archive measures the gzip of it.
//
// Permissions travel beside the tree, not on it. A file on the volume is read by an
// execution that may run as another user, so what is stored is widened to be readable
// by anyone - which loses the source's own mode. modes.v1, written beside <entry>/root,
// records every path whose permissions differ from DefaultFileMode (0644) or
// DefaultDirMode (0755); the restore sets those defaults on everything it writes -
// including the declared directory itself, which openDeclaredRoot makes rather than the
// walk - and then applies the recorded ones, deepest first, so narrowing a directory
// cannot shut the restore out of what is still inside it. Set, not passed to MkdirAll
// and OpenFile and left there: a mode given at creation is filtered by the restoring
// process's umask, so under 0077 a tree cached at an ordinary 0755/0644 would come back
// private to whoever restored it, and the manifest records only what differs from the
// defaults so the defaults have to be applied rather than assumed. Without this a key
// saved 0600 came back 0666 and ssh refused it, where the archive backend carries each
// mode in its tar header - the same workflow behaving differently by backend. The file
// is versioned in its name so an older agent meeting a newer entry simply ignores it
// and restores at the defaults, and Store.OpenEntry therefore roots at the entry, with
// the tree inside it. It is streamed and bounded by the entry limit (DecodeModesFrom,
// with one record capped at maxModeRecordBytes, records counted as read rather than as
// kept so that a repeated or malformed one still spends the budget, and the paths
// retained capped at maxModeTotalBytes because a count alone leaves their length free):
// it is read before any copy limit applies, and the inbox an entry is built in is
// mounted into the step's own container - so a workflow can commit a manifest of any
// size, which every later execution restoring that key would otherwise read into memory
// whole. The save records under the same cap (Recorder) and streams the manifest out
// rather than encoding it into a buffer first: the walk feeding it is bounded only by
// the entry limit, so 500,000 paths of up to PATH_MAX is gigabytes of strings held in
// the step's own container, where being killed for it fails the step. Matching the
// decode's number is what makes dropping the excess free - a record past it is one
// every restore would discard anyway.
//
// Symlinks are carried, and never followed on the way out. A restore writes through an
// os.Root opened on each declared path, so a symlink already sitting there cannot
// redirect a write outside it - the entry may have been written by another workflow.
// The links inside an entry are recreated as links: node_modules/.bin is entirely
// symlinks, and dropping them would restore an incomplete tree that still answers as an
// exact hit, so the step would never repair it.
//
// A volume that will not take an entry falls back to the object store. The open-time
// probe cannot predict a volume filling mid-copy, so saveToVolume reports whether it
// settled the save. What falls through is decided by whether the other backend would
// answer differently - and both limits do: the size, because the volume weighs the tree
// where the archive weighs the gzip of it, and the count, because the volume counts
// every directory where the archive's walker emits only files and links. A failed copy
// or commit falls back too. What keeps a fallback from publishing an entry no restore
// would accept is that the archive path enforces cacheMaxEntries on its own count
// before asking for a grant: an archive over it would store happily and then be refused
// by every restore of it, under a key that is immutable, so the step would reinstall
// every execution until it expired.
//
// # Configuration
//
// Step dependency caches can be kept on a shared ReadWriteMany volume instead of whole
// in the object store, via TESTKUBE_STEP_CACHE_VOLUME_CLAIM (an existing RWX PVC, which
// must exist in every execution namespace; empty disables it),
// TESTKUBE_STEP_CACHE_VOLUME_MOUNT_PATH (where the agent mounts it for the expiry
// sweep), TESTKUBE_STEP_CACHE_VOLUME_RETENTION_DAYS (default 7) and
// TESTKUBE_STEP_CACHE_VOLUME_SWEEP_INTERVAL (default 1h). Helm exposes all four as
// stepCacheVolume.* on both the testkube-api and testkube-runner charts, because both
// build execution pods, and both mount the claim on their own Deployment so the
// leader-gated sweep can reach it in a runner-only installation too. The volume and its
// sweep run in every mode; the claim alone enables them. Retention shorter than the
// earlier of STORAGE_CACHE_EXPIRATION and STORAGE_EXPIRATION is raised to it at
// startup, with a warning, rather than used as given. A runner owns no bucket and
// cannot read its lifecycle, so it is told one through
// stepCacheVolume.objectExpirationDays (default 1), which reaches
// STORAGE_CACHE_EXPIRATION; the api chart uses storage.cacheExpiration as before. With
// nothing expiring a pointer the sweep still runs on the configured retention, and the
// agent warns once that a swept entry may leave a pointer behind it.
package volume
