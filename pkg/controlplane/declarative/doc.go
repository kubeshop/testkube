// Package declarative holds the rules for Control Plane resources: entities
// that live only in the Control Plane, declared in CRD-shaped YAML, applied
// with `testkube apply -f`, exported with `testkube get <kind> <name> -o yaml`
// and removed with `testkube delete <kind> <name>`. Each entity family has an
// API group and a schema package of its own under api/ (such as
// ai.testkube.io in api/ai/v1alpha1), with CRDs generated into k8s/crd.
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
// Adding a kind takes a types package (or a type in an existing group), an
// entry in kinds in registry.go (the plural resource must be unique across
// groups, since it is the endpoint path segment), cases in Normalize,
// PrepareForApply and StatusOf, fixtures under testdata/, and the Control
// Plane endpoint. The CLI's get and delete subcommands are generated from the
// registry.
//
// The registry is in apply order: a kind comes after every kind it refers to.
// SortForApply puts MCP servers and LLM providers before the AI Agents that
// link to them, and AI Agents before the Test Bundles that delegate to them,
// so one input applies whatever order it lists them in.
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
// name looks like a credential (IsCredentialField, which includes sig, the
// signature of an Azure SAS URL) and a URL with user info. Such an error
// echoes neither the value nor the refused name, which may hold a secret
// itself: it is reported on the URL or on the whole header or query map.
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
// An agent's Skills belong to it: applying a skills list replaces the whole
// list, in manifest order, matched by name, so a Skill left out is deleted,
// and skills: [] removes them all. An agent that leaves skills out keeps its
// Skills as stored, as it keeps its built-in server link, so applying a
// manifest written before Skills existed does not delete the ones added in
// the dashboard. AIAgentSpec.Skills is therefore not omitempty: nil (absent
// or null) and an empty list mean different things. Each
// follows the Agent Skills spec's limits (MaxSkillNameLength and the others,
// in skill.go), and the content of those always included is at most
// MaxAlwaysIncludedSkillsLength characters together, since every prompt
// carries it; the others are read on demand.
//
// # Test Bundles
//
// A Test Bundle is environment-scoped. Its metadata.name is the slug the
// Control Plane gave it, which never changes: lowercase letters, digits, '-'
// and '_', at most 59 characters (MaxTestBundleNameLength), since it labels
// the bundle's workflow and sandbox. A slug made in the dashboard can hold
// '_', so unlike an AI resource's name it is not always a DNS-1123 label.
//
// The manifest declares the bundle's definition only: display name,
// description, labels, the Git source its tests are read from, its Skills, its
// Attached Context and the AI Agents it may delegate to. What authoring produces (test files,
// chats, the sandbox, the shared workflow and the GitHub repository set up for
// chats) stays in the Control Plane; the workflow's name and chat activity are
// exported in status and never applied. Deleting a bundle is the dashboard's
// delete: a soft delete that also ends its chats and sandbox.
//
// Skills belong to the bundle, keyed by name and kept in manifest order:
// applying a skills list replaces the whole list, so a Skill left out is
// deleted, and skills: [] removes them all. A bundle that leaves skills out
// keeps its Skills as stored, as an AI Agent does, so a manifest without them
// does not delete the ones added in the dashboard; TestBundleSpec.Skills is
// therefore not omitempty either. They follow
// the Agent Skills spec's limits (skill.go) and are always included, so
// together they hold at most MaxAlwaysIncludedSkillsLength characters of
// content, since every authoring prompt carries all of them. A Git source URI
// must not carry credentials, which Normalize reports without echoing the URI.
//
// contexts is the bundle's Attached Context: guidance its authoring sandbox
// holds for the AI to read, apart from its Skills. It is kept in manifest
// order and replaced as a whole as skills is, and left as stored when
// omitted, so TestBundleSpec.Contexts is not omitempty either. Each entry
// needs a title and content and nothing more, as in the dashboard: titles may
// repeat, so an entry has no key, and applying a list writes it position by
// position.
//
// It must be a URL with a scheme and host, or Git's scp-like [user@]host:path,
// whose path may hold an '@' (team/repo@v2.git) unless, with no user, it comes
// before the path's first '/', where host:secret@... reads as
// user:password@host. Anything else, such as a URL url.Parse cannot read, or
// one whose query url.ParseQuery cannot read, is refused, since its
// credentials could not be checked.
//
// aiAgents names AI Agents by metadata.name and is sorted, as the Control
// Plane exports it. Omitted, a new bundle starts with the environment's
// defaults for new bundles and an existing one keeps its own; an empty list
// delegates to none. The field is a pointer so the two reach the Control Plane
// apart.
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
// every document before sending any, sends them in registry order
// (SortForApply), and keeps going past a failed PUT, reporting all failures
// together.
//
// Not yet done: GitOps sync of these CRDs (a generic SyncService arm and a
// reconciler), and Control Plane enforcement of read-only GitOps-owned
// resources.
package declarative
