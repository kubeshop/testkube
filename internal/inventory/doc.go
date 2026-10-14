// Package inventory is the Agent side of AgentInventoryService: it publishes a
// snapshot of the resource kinds the Agent can watch in its cluster to the
// Control Plane, which caches it to render the TestTrigger resourceRef
// picker. It is
// cluster-environment metadata, not Testkube objects - those travel through
// internal/sync.
//
//   - controller runs the push loop (startup, an hourly safety net, and
//     debounced events from a CRD informer) and the CRD informer itself.
//   - grpc is the Control Plane client.
//
// # When it runs
//
// cmd/api-server starts it only when config.ShouldPushClusterInventory
// agrees, which requires all of:
//
//   - test triggers enabled. The picker is the only consumer, and the CRD
//     informer needs cluster-scoped CRD read (the crd-reader ClusterRole in
//     the testkube-api chart). Without that permission the informer logs a
//     failed watch on every retry, so DISABLE_TEST_TRIGGERS turns it off
//     rather than leave that noise on an install with nothing to show it to.
//   - a Control Plane connection. A standalone agent serves discovery from
//     its own API and has nowhere to push.
//   - the listener capability. The Control Plane rejects pushes from any
//     other agent, and a runner-only deployment has no CRD RBAC to watch with.
package inventory
