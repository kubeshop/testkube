// Package declarative holds the rules for Control Plane resources: entities
// that live only in the Control Plane (Insights boards, AI Agents with their
// MCP servers and LLM providers; test bundles later) declared in CRD-shaped
// YAML. Each entity family has an API group and a schema package of its own:
// insights.testkube.io (api/insights/v1alpha1) for boards and ai.testkube.io
// (api/ai/v1alpha1) for the AI kinds, with testbundles.testkube.io to come.
// Shared status types are in api/common/v1.
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
// The registry is in apply order: a kind comes after every kind it refers to.
// SortForApply puts MCP servers and LLM providers before the AI Agents that
// link to them, so one input applies whatever order it lists them in.
//
// # AI resources
//
// AI Agents, MCP servers and LLM providers are environment-scoped. Their
// metadata.name is a DNS-1123 label, so it also names a custom resource; the
// Control Plane stores it as the entity's slug, and ProposeName makes one
// from a display name for an entity created without one. The stored models
// live in the Control Plane, which maps them itself; what both ends must
// agree on is here.
//
// A manifest never carries a secret. A key is named by an Environment
// Credential in credentialReference (secret://<name>, or vault://<path> for
// an LLM provider), and Normalize refuses a header or query parameter whose
// name looks like a credential (IsCredentialField) and a URL with user info.
// The Control Plane exports every header and query parameter value as
// RedactedValue, which applied back keeps the stored value, so the
// placeholder is accepted under any name. A
// shared-key link without a credentialReference keeps the key set in the
// dashboard, which is never exported.
//
// An agent links to an MCPServer by its metadata.name, and to the built-in
// Testkube MCP server as "testkube", a name no MCPServer may take. An agent
// that leaves the built-in server out keeps its link to it as stored (the
// default tools, for a new agent), so applying an older manifest does not
// cut the agent off from Testkube. Access roles and resource groups are sorted,
// as the Control Plane returns them.
//
// # Normalization
//
// Normalize validates an object and fills in its defaults, in place. It is
// idempotent - normalizing a normalized object changes nothing - which is what
// lets an exported object be re-applied without a diff. Generated values are
// deterministic for the same reason: a report filter without an id gets
// <report id>-<index>, never a random one.
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
// envelope (it pages only when asked to). Status, and with it GitOps ownership, is never taken from
// a request body.
package declarative
