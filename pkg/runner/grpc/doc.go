// Package grpc is the runner's client for the executions the control plane hands it.
//
// # Polling
//
// The client polls GetExecutionUpdates, tuned by the RUNNER_* settings in internal/config:
//
//   - RUNNER_POLL_INTERVAL is a latency floor rather than a throughput cap, because one poll
//     returns a whole batch.
//   - RUNNER_CALL_TIMEOUT bounds each call.
//   - RUNNER_POLL_MAX_BACKOFF caps the retry delay after a failed poll.
//   - RUNNER_START_CONCURRENCY is how many executions from one batch start at once.
//   - RUNNER_POLL_STALE_AFTER is how long without a successful poll before /ready reports the
//     runner unready (Healthy).
//
// Keep RUNNER_POLL_STALE_AFTER above RUNNER_CALL_TIMEOUT + RUNNER_POLL_MAX_BACKOFF, so one slow
// poll cannot flap readiness.
//
// # Health and readiness
//
// /health answers unconditionally and is what the startup and liveness probes use. /ready
// evaluates the registered checks, this client's included, and returns 503 with the names of the
// failing ones. The api and runner charts point their readiness probes at /ready. Liveness
// deliberately stays on /health: a stalled poll loop usually means the control plane is
// unreachable, and restarting every runner over that is worse than the stall.
package grpc
