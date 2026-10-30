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
//
// # Identity tokens
//
// With a signing key (IDENTITY_TOKEN_PRIVATE_KEY_FILE), GetIdentityToken mints identity_token() tokens
// (pkg/identitytoken). Nothing else on this gRPC port authenticates the caller, so dispatch hands each
// execution an execution token derived from the key, the runner passes it to the execution's pods as their
// api-key, and GetIdentityToken mints a token only for the execution whose token it is. Without the key the
// execution token stays empty, the pods keep the agent key as before, and GetIdentityToken returns
// FailedPrecondition.
package controlplane
