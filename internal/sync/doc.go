// Package sync contains the implementation of a Sync (or GitOps) capability for a Testkube Agent:
// the agent side of syncing Kubernetes resources into the Control Plane.
//
// Reconcilers live in the controller package, one per syncable kind, and the Control Plane client
// lives in the grpc package. The syncable kinds are TestWorkflow, TestWorkflowTemplate,
// TestTrigger, WorkflowTrigger, Webhook and WebhookTemplate. Adding a kind means changing the
// controller and grpc packages, the SyncService proto and the Control Plane together.
//
// The sync controllers are registered in cmd/api-server/main.go behind
// proContext.CloudStorageSupportedInControlPlane and GITOPS_KUBERNETES_TO_CLOUD_ENABLED (default
// false), so they only run for a GitOps-persona agent connected to a Control Plane that reports
// cloud storage support.
//
// # Resource ownership contract
//
// A synced resource in the Control Plane is exclusively owned by one GitOps agent, so that agents
// syncing from different namespaces cannot silently overwrite each other's resources. The
// contract spans four layers, and a change to one usually needs a matching change in the others:
//
//  1. Schema: Syncable.gitOpsOwner in api/v1/testkube.yaml, a GitOpsOwner holding the
//     authoritative agentId plus a display-only agentName that may be stale after a rename. Run
//     `make generate-openapi` after editing.
//  2. Wire: the Control Plane rejects a sync from a non-owning agent with the gRPC status
//     codes.FailedPrecondition. That code is reserved for ownership conflicts on the sync API; do
//     not reuse it there for other validation failures, or agents will misreport the cause.
//  3. Agent translation: translateError in grpc/errors.go maps FailedPrecondition onto the
//     ErrOwnershipConflict sentinel in errors.go, keeping the original status in the error chain
//     because its message names the current owner. Callers match on the sentinel so that no layer
//     above the client depends on gRPC.
//  4. Reconciliation: terminalOnRejection in controller/errors.go wraps a rejection (an ownership
//     conflict or an invalid resource, see syncagent.IsRejection) in reconcile.TerminalError so
//     controller-runtime stops requeueing something no retry can fix, while every other error
//     stays retryable with backoff. Rejections are deliberately not logged there:
//     controller-runtime already logs whatever a reconciler returns, along with the kind and name
//     of the resource.
//
// skipRejectedResource in cmd/api-server/superagentmigration.go applies the same rule outside the
// reconcilers, during SuperAgent migration. That migration retries sync failures forever by
// design, so a rejection has to break the loop or a single unowned or invalid resource wedges the
// migration indefinitely.
//
// The Control Plane side lives in testkube-cloud-api: it persists the owner on the six syncable
// tables and enforces it in internal/sync/ownership.go, on every Upsert and Delete path.
// Enforcement is opt-in per organization (the gitOpsOwnership setting), so with it off an agent
// never sees a conflict. Ownership is transferred from Git by setting the testkube.io/gitops-owner
// annotation on the Kubernetes resource to the new owner's agent ID. The agent sends the
// annotation with the resource, and the Control Plane reads it, so the agent needs no code for it.
// The agent name is display-only and never authorizes anything.
package sync
