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
//     request carries the environment-id header only for an Environment other than the one of
//     the agent token; without it the Control Plane uses the Environment of the agent token, and
//     an Environment the agent may not use is refused with codes.PermissionDenied
//     (ErrEnvironmentForbidden on the agent, see step 3).
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
// # Namespaces and Environments
//
// GITOPS_NAMESPACES lists the namespaces the controller manager watches, comma separated. An
// entry is ns, ns=<envId> or *. Empty means TESTKUBE_NAMESPACE. config.ParseGitOpsNamespaces
// parses it. The testkube-runner chart renders it from gitops.namespaces (a list, or a map of
// namespace to Environment id) and gitops.watchAllNamespaces, and sets
// GITOPS_EXECUTE_IN_SOURCE_NAMESPACE=true with it.
//
// With sync on, the agent exits at startup when an envId is not an Environment of the agent.
// Without the sync-multi-namespace capability it also exits when a namespace maps to another
// Environment than the token one, when several namespaces sync into one Environment, or with *
// (config.ValidateGitOpsNamespaces). One namespace in the token Environment passes without the
// capability, as before.
//
// Each store call takes a Target (target.go): the namespace of the resource and the Environment
// that config.GitOpsNamespaces.EnvironmentID maps it to, empty for the Environment of the agent
// token (ProContext.TokenEnvID). The grpc client puts the namespace on every update and delete:
// each *Id message of DeleteRequest and the UpdateOrCreateRequest have an optional namespace
// field. When it is empty the Control Plane falls back to the default namespace of the agent for
// a delete, and to the namespace inside the payload for an update. The client sends the
// environment-id header only for another Environment, and adds the label
// testkube.io/source-namespace (SourceNamespaceLabel) to a copy of the resource. The Control
// Plane advertises that it reads these fields and the header with the sync-multi-namespace
// capability (capabilities.CapabilityMultiNamespaceSync), which the agent exposes as
// ProContext.MultiNamespaceSyncSupportedInControlPlane. Without it an agent syncs one namespace
// per Environment.
//
// The owner key stays the pair (agentId, namespace), so one agent that watches several
// namespaces owns the resources of each namespace separately. A resource that a per-namespace
// agent synced before keeps that owner until its testkube.io/gitops-owner annotation names the
// global agent, or the Control Plane transfers it. A resource synced before the namespace was
// recorded has an empty namespace; the Control Plane decides how it matches a namespaced owner.
//
// The Control Plane keys a resource by its name in an Environment, so a name must be
// unique within an Environment across every namespace that syncs into it. The controller package
// refuses the loser of two resources of the same kind and name with ErrNameCollision, a terminal
// error, and documents the rule in controller/doc.go.
//
// The manager caches TestWorkflowExecution only in TESTKUBE_NAMESPACE. It runs with leader
// election unless LEADER_ELECTION_DISABLED is set, with the lease
// testkube-controllers-<APIServerFullname> in TESTKUBE_NAMESPACE.
// GITOPS_EXECUTE_IN_SOURCE_NAMESPACE makes a synced workflow run in its metadata.namespace when
// execution.additionalNamespaces (TESTKUBE_EXECUTION_NAMESPACES) allows it, see
// pkg/testworkflows/executionworker/kubernetesworker/doc.go.
//
// The Control Plane side lives in testkube-cloud-api: it persists the owner on the six syncable
// tables and enforces it in internal/sync/ownership.go, on every Upsert and Delete path.
// Enforcement is opt-in per organization (the gitOpsOwnership setting), so with it off an agent
// never sees a conflict. Ownership is transferred from Git by setting the testkube.io/gitops-owner
// annotation on the Kubernetes resource to the new owner's agent ID. The agent sends the
// annotation with the resource, and the Control Plane reads it, so the agent needs no code for it.
// The agent name is display-only and never authorizes anything. The same rules are in
// testkube-cloud-api's internal/sync/doc.go; the header and the namespace fields are read in its
// internal/sync/grpc. A change to a rule here is a change there.
package sync
