// Package controlplane is the agent-side control plane of a standalone installation: it schedules
// executions and serves them to the runner over gRPC.
//
// # Execution dispatch
//
// Dispatch is bounded by EXECUTION_DISPATCH_BATCH_SIZE, the number of executions one
// GetExecutionUpdates poll hands to the runner. A dispatched execution is re-offered after
// EXECUTION_REDISPATCH_AFTER if the runner never acknowledged it, and is failed explicitly after
// EXECUTION_START_TIMEOUT by the reaper (reaper.go), which runs every EXECUTION_REAPER_INTERVAL.
// The defaults are in internal/config.
//
// EXECUTION_START_TIMEOUT must stay above EXECUTION_REDISPATCH_AFTER, or the reaper fails
// executions before they are retried; the server clamps it if it is not. Raising the batch size
// trades a bounded poll for a larger one: the bound is what keeps a burst of workflows sharing a
// cron minute from exceeding the runner's call deadline.
package controlplane
