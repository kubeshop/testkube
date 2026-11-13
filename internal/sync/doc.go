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
//     authoritative agentId, the namespace the agent synced the resource from, and a
//     display-only agentName that may be stale after a rename. The owner key is the pair
//     (agentId, namespace); the namespace is empty for resources synced before it was recorded.
//     Run `make generate-openapi` after editing.
//  2. Wire: the Control Plane rejects a sync from a non-owning agent with the gRPC status
//     codes.FailedPrecondition. That code is reserved for ownership conflicts on the sync API; do
//     not reuse it there for other validation failures, or agents will misreport the cause. A
//     request may carry the environment-id header: without it the Control Plane uses the
//     Environment of the agent token, and an Environment the agent may not use is refused with
//     codes.PermissionDenied.
//  3. Agent translation: translateError in grpc/errors.go maps FailedPrecondition onto the
//     ErrOwnershipConflict sentinel in errors.go, keeping the original status in the error chain
//     because its message names the current owner, and PermissionDenied onto
//     ErrEnvironmentForbidden. Callers match on the sentinel so that no layer above the client
//     depends on gRPC.
//  4. Reconciliation: terminalOnRejection in controller/errors.go wraps a rejection (an ownership
//     conflict, an invalid resource, a forbidden Environment or a name collision, see
//     syncagent.IsRejection) in reconcile.TerminalError so controller-runtime stops requeueing
//     something no retry can fix, while every other error stays retryable with backoff. Rejections
//     are deliberately not logged there: controller-runtime already logs whatever a reconciler
//     returns, along with the kind and name of the resource.
//
// skipRejectedResource in cmd/api-server/superagentmigration.go applies the same rule outside the
// reconcilers, during SuperAgent migration. That migration retries sync failures forever by
// design, so a rejection has to break the loop or a single unowned or invalid resource wedges the
// migration indefinitely.
//
// The sync API carries the namespace explicitly: each *Id message of DeleteRequest and the
// UpdateOrCreateRequest have an optional namespace field. When it is empty the Control Plane
// falls back to the agent's default namespace for a delete, and to the namespace inside the
// payload for an update. The Control Plane advertises that it reads these fields and the
// environment-id header with the sync-multi-namespace capability
// (capabilities.CapabilityMultiNamespaceSync), which the agent exposes as
// ProContext.MultiNamespaceSyncSupportedInControlPlane. Without it an agent syncs one namespace
// per Environment.
//
// GITOPS_NAMESPACES sets the namespaces that the controller manager watches, comma separated. An
// entry is ns, ns=<envId> (sync ns into that Environment) or * (every namespace); empty means
// TESTKUBE_NAMESPACE. config.ParseGitOpsNamespaces parses it. With GitOps sync on, the agent
// exits at startup when an envId is not an Environment of the agent, or when a namespace maps to
// an Environment other than the one of the agent token and the Control Plane lacks
// sync-multi-namespace (config.ValidateGitOpsNamespaces). The manager runs with leader election
// unless LEADER_ELECTION_DISABLED is set, so one replica syncs; the lease is
// testkube-controllers-<APIServerFullname> in TESTKUBE_NAMESPACE. The TestWorkflowExecution
// controller still handles only TESTKUBE_NAMESPACE.
//
// A store call takes a Target (target.go): the namespace of the resource and the Environment that
// config.GitOpsNamespaces.EnvironmentID maps it to, empty for the Environment of the agent token.
// The grpc client puts the namespace on every update and delete. It sends the environment-id
// header only for an Environment other than the one of the agent token, because a Control Plane
// without sync-multi-namespace refuses the header. It adds the label SourceNamespaceLabel
// (testkube.io/source-namespace) to a copy of each resource it sends, so the Kubernetes object in
// the cache stays unchanged. The Control Plane keys a resource by its name in an Environment, so
// the controller package refuses a second resource with the same name from another namespace of
// the same Environment (ErrNameCollision); the controller package documents the rule.
//
// The Control Plane side lives in testkube-cloud-api: it persists the owner on the six syncable
// tables and enforces it in internal/sync/ownership.go, on every Upsert and Delete path.
// Enforcement is opt-in per organization (the gitOpsOwnership setting), so with it off an agent
// never sees a conflict. Ownership is transferred from Git by setting the testkube.io/gitops-owner
// annotation on the Kubernetes resource to the new owner's agent ID. The agent sends the
// annotation with the resource, and the Control Plane reads it, so the agent needs no code for it.
// The agent name is display-only and never authorizes anything.
package sync
