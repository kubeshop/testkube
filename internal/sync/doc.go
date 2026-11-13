// Package sync contains the implementation of a Sync (or GitOps) capability for a Testkube Agent:
// the agent side of syncing Kubernetes resources into the Control Plane.
//
// Reconcilers live in the controller package, one per syncable kind, and the Control Plane client
// lives in the grpc package. The syncable kinds are TestWorkflow, TestWorkflowTemplate,
// TestTrigger, WorkflowTrigger, Webhook and WebhookTemplate, each with its own reconciler and its
// own arm of the SyncService proto; adding one of these means changing the controller and grpc
// packages, the SyncService proto and the Control Plane together.
//
// The sync controllers are registered in cmd/api-server/main.go behind
// proContext.CloudStorageSupportedInControlPlane and GITOPS_KUBERNETES_TO_CLOUD_ENABLED (default
// false), so they only run for a GitOps-persona agent connected to a Control Plane that reports
// cloud storage support.
//
// # Control Plane resources
//
// Control Plane resources, the kinds that live only in the Control Plane and are declared in the
// registry of pkg/controlplane/declarative, share one reconciler and one
// SyncService arm (ControlPlaneResource, named by the kind's resource and the object's name). The
// reconciler (controller.NewControlPlaneResourceSyncControllers) checks each object with the shared
// Normalize before sending it, so an invalid one is a terminal error reported in the cluster and is
// never sent. It sends the object as declarative.ApplyPayload makes it for testkube apply: without
// status, and with its annotations, so testkube.io/gitops-owner and testkube.io/no-gitops-sync work
// as they do for the other kinds. Syncing another kind means adding it to
// syncedControlPlaneResources in the controller package, its scheme in main.go, read access in the
// Helm charts, and its arm in the Control Plane's handler; the proto stays as it is.
//
// Each kind also needs the Control Plane to advertise its family's capability, so a Control Plane
// that does not serve the kind never gets it: controlplane-resources for Insights Boards,
// controlplane-resources-ai for MCP servers, LLM providers and AI Agents, and
// controlplane-resources-testbundles for Test Bundles (capabilities.CapabilityControlPlaneResources
// and the two beside it). The kinds are registered in the order they reference each other, MCP
// servers before the AI Agents that link them and AI Agents before the Test Bundles that name
// them, so a first sync needs fewer retries. A resource naming one the Control Plane does not have
// yet is retried with backoff, not refused, as what it names may be synced next. A
// kind whose CRD is not installed, that the agent may not list and watch in its namespace, or
// whose access the agent may not check (a cluster that forbids SelfSubjectAccessReviews) is
// skipped with a warning: a controller whose cache cannot sync would stop the manager, and the
// agent with it. The Helm charts install the CRDs (testkube-crds) and grant the read access (the
// runner chart's watcher role when gitops.enabled, and the testkube-api chart's
// controlplaneresources role).
//
// While a GitOps agent owns a Control Plane resource, it is read-only everywhere else in the
// Control Plane (the dashboard, the MCP tools, and testkube apply and delete), so it cannot drift
// from Git. For an AI Agent that includes its roles, resource groups and own Skills, and for a Test
// Bundle its Attached Context, own Skills and AI Agents. A synced AI Agent is never private: the
// Control Plane refuses one that is. Deleting the custom resource deletes it in the Control Plane,
// unless another agent owns it: that delete is an ownership conflict, and the resource is left as
// it is. Deleting a TestBundle custom resource deletes the Test Bundle as the dashboard does: it is
// soft-deleted, and its chats and their sandbox end with it.
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
