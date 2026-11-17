// Package paralleldata exposes what the workers of a parallel step published to the
// expression language, so that a later step can aggregate per-worker results:
//
//	{{ parallel("shard", 0).outputs.duration }}
//
// A worker writes to its outputs directory as any step does. Its own init scans the
// directory and prints an outputs instruction, which the parallel step reads from the
// worker's status stream. When the workers end, the parallel step prints one hint per
// step, named after the step id, holding every worker's status and outputs.
//
// The hint is stored in the state of the pod running the parallel step, where the later
// steps read it. It is not saved on the execution record: a workflow that wants to publish
// an aggregate writes it to its own outputs directory.
//
// Sensitive values never leave the worker: its init replaces them with withheld markers
// before printing the instruction, so a consumer that refuses markers keeps refusing them.
//
// # Resolving a reference
//
// parallel("<step id>", <index>) returns {index, status, outputs} for a worker of an earlier
// parallel step. The ref is the step's id, explicit or derived from its name, and
// ProcessParallel passes it to the toolkit as --id. A step with neither an id nor a name
// records nothing. The index is required, and an index outside the expected worker count is
// an error. A worker that never started resolves with no status and no outputs.
//
// # Transport
//
// StatusNotifications forwards a worker's outputs instructions only when
// StatusNotificationsOptions.Outputs is set. Leave it off for other consumers: the services
// step reads every notification as a readiness change. StatusNotifications sends with
// SendWithContext: the parallel step stops reading on fail-fast, and a blocked send would
// keep the watch and its registry connection alive.
//
// The toolkit's parallel.go merges the instructions per worker (workerOutputs, through
// executiondata.MergeOutputs) and prints the hint paralleldata.<id> on every exit path,
// failures included. A worker whose monitoring ended without a terminal status, for example
// because its watch failed, is recorded as aborted (finishedStatus). The cleanup destroys
// it, so running would never become true.
//
// The hint is pod-local (init state, read by data.ParallelDataMachine), and it is wired next
// to every data.ExecutionDataMachine use. It is not exposed through execution(). The name
// avoids "parallel", the output that reports worker status to the UI and runner. It is also
// in data.Expression, so step conditions and result expressions can read it, unlike
// execution(), which needs the Control Plane.
//
// # Worker specs
//
// Worker specs are resolved in the parent pod, through resolveWorkerSpec in parallel.go,
// with NewScopedMachine rather than the strict machine. It resolves the groups that finished
// in the parent, and leaves any reference the parent does not know, or one that a worker's
// own parallel step shadows, for the pod that owns it. Each nested parallel body is a scope
// of its own, because it runs in pods of its own. A level's declarations shadow the parent's
// groups in that level and the levels nested in it, never in the levels around it. The
// levels are resolved outermost first:
//
//  1. The step names are resolved and saved on the spec, because the worker's processor
//     derives ids from the resolved names: "Shard {{ index }}" is shard_0 in worker 0.
//  2. The level's ids are derived, with the processor's _1 suffixes, across a body's setup,
//     steps and after.
//  3. The level is simplified with its nested bodies detached.
//  4. The resolver recurses into each body.
//
// Worker specs are resolved twice, in buildSingleWorkerSpec and again in ExecuteWorker. While
// a name still holds an expression in the first pass, that level and the levels in it defer
// parallel() to the second pass, which derives the id from the name as it is. Keep
// parallel() out of the base machines: whichever machine resolves it first wins.
package paralleldata
