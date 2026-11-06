// Package declarative holds the rules for Control Plane resources: entities
// that live only in the Control Plane, declared in CRD-shaped YAML, applied
// with `testkube apply -f`, exported with `testkube get <kind> <name> -o yaml`
// and removed with `testkube delete <kind> <name>`. Each entity family has an
// API group and a schema package of its own under api/ (such as
// insights.testkube.io in api/insights/v1alpha1), with CRDs generated into
// k8s/crd. Shared status types are in api/common/v1.
//
// It is a contract between three consumers, which must all go through it
// rather than reimplement it:
//
//   - the CLI (testkube apply/get/delete), which validates locally before it
//     sends anything;
//   - the GitOps agent, which syncs the same objects as custom resources;
//   - the Control Plane (testkube-cloud-api internal/declarative), which
//     validates and normalizes every PUT with Normalize, and maps an object
//     to and from its stored form (BoardToStored, BoardFromStored).
//
// A disagreement between them would make an object the CLI accepts fail in
// the Control Plane, or make an export fail to re-apply.
//
// # Registry
//
// Kinds lists every supported kind with its plural resource name (the path
// segment of the generic endpoints), its CLI names and its scope. Decode reads
// multi-document YAML and dispatches each document on apiVersion and kind.
//
// Adding a kind takes a types package (or a type in an existing group), an
// entry in kinds in registry.go (the plural resource must be unique across
// groups, since it is the endpoint path segment), cases in Normalize,
// PrepareForApply and StatusOf, fixtures under testdata/, and the Control
// Plane endpoint. The CLI's get and delete subcommands are generated from the
// registry.
//
// # Normalization
//
// Normalize validates an object and fills in its defaults, in place. It is
// idempotent - normalizing a normalized object changes nothing - which is what
// lets an exported object be re-applied without a diff. Generated values are
// deterministic for the same reason: a report filter without an id gets
// <report id>-<index>, never a random one. TestFixtures_Valid checks this on
// every fixture in testdata/valid. Board report params go through
// pkg/mcp/boards, so a dashboard change ported there flows through here too.
//
// Normalize gives a new report the dashboard's creation defaults, which for
// some params differ from what the dashboard renders when a stored report
// lacks them (a pass-fail report without a duration shows a week, but is
// created with a month). BoardFromStored therefore writes those rendered
// defaults into an export, and Normalize accepts them, so stored → export →
// apply leaves a report showing what it showed.
//
// # Metadata
//
// A board keeps its metadata.labels: Normalize checks them against the
// Kubernetes label rules, BoardToStored stores them with the board and
// BoardFromStored returns them, so an export carries the labels that were
// applied. Annotations are not stored; they are sent on apply, and the Control
// Plane ignores them.
//
// # Wire format
//
// The client package talks to the Control Plane's generic, name-keyed
// endpoints:
//
//	GET|PUT|DELETE /organizations/{org}[/environments/{env}]/resources/{resource}/{name}
//	GET            /organizations/{org}[/environments/{env}]/resources/{resource}
//
// The environment segment is present only for environment-scoped kinds. PUT
// takes the object as JSON, stripped by PrepareForApply, and returns it as
// stored. GET returns the object with status filled in, and the list returns
// every object of the kind as {"elements": [...]}, the Control Plane's list
// envelope (it pages only when asked to). Status, and with it GitOps
// ownership, is never taken from a request body. The client refuses an
// organization-scoped kind with a tkcapi_ API token before sending anything,
// as pkg/mcp/api.go does for boards.
//
// # CLI
//
// The CLI (cmd/kubectl-testkube/commands/controlplaneresources) validates
// every document before sending any, and keeps going past a failed PUT,
// reporting all failures together.
//
// Not yet done: GitOps sync of these CRDs (a generic SyncService arm and a
// reconciler), and Control Plane enforcement of read-only GitOps-owned
// resources.
package declarative
