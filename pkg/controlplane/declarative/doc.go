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
// SortForApply puts MCP servers, LLM providers and Skills before the AI Agents
// that link to them, so one input applies whatever order it lists them in.
//
// # AI resources
//
// AI Agents, MCP servers, LLM providers and Skills are environment-scoped. Their
// metadata.name is a DNS-1123 label, so it also names a custom resource; the
// Control Plane stores it as the entity's slug, and ProposeName makes one
// from a display name for an entity created without one. The stored models
// live in the Control Plane, which maps them itself; what both ends must
// agree on is here.
//
// Credentials follow the rules of the dashboard and the REST API, as Test
// Workflows do: a manifest may carry them. An LLM provider's headers, query
// parameters and URLs are taken as written, an X-Api-Key header or a key in a
// query included. A credentialReference names an Environment Credential
// (secret://<name>, or vault://<path> for an LLM provider) and is the way to
// keep a key out of Git, so it must be a name, not a key itself. The Control
// Plane exports every header and query parameter value as RedactedValue, as
// its REST API returns them, and keeps the stored value for one applied back
// that way. A URL or a credentialReference refused as malformed is not echoed
// in the error, since it may hold a secret, and errors reach terminal and CI
// logs. A shared-key link without a credentialReference keeps the key set in
// the dashboard, which is never exported.
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
// # Skills
//
// A Skill is a Skill of the environment's library, which AI Agents share; an
// agent's own Skills stay in its spec.skills. It follows the Agent Skills spec
// (https://agentskills.io/specification) key for key: metadata.name is the
// SKILL.md name, spec holds description, license, compatibility, metadata and
// allowed-tools as the frontmatter does, and content is the Markdown body.
// Settings of Testkube's own are under spec.testkube (alwaysInclude,
// addToNewTestBundles); an empty block is dropped, as both off is the
// default. Its metadata.name is both a DNS-1123 label and a spec name, so it
// has no consecutive hyphens and at most 63 characters. The spec's scripts/,
// references/ and assets/ folders are not declared yet.
//
// A Skill, and an AI Agent's own Skill, may instead read a real SKILL.md from
// a ConfigMap with skillFrom.configMapKeyRef, so the file kept in Git is the
// source. The SKILL.md then gives every field but spec.testkube (an own
// Skill's alwaysInclude stays in the manifest), so writing one beside skillFrom
// is refused, and its name must be the Skill's. Only a GitOps runner can read
// the ConfigMap: ResolveSkillSources, given a reader of its cluster, replaces
// skillFrom with what the SKILL.md holds before Normalize, so the Control Plane
// gets the Skill as if written inline. Normalize refuses a skillFrom left in
// place, which is what testkube apply sees. ParseSkillMarkdown reads the
// file: frontmatter keys outside the spec are ignored, since the same file
// often serves other agent tools; metadata values must be strings; and
// allowed-tools is a space-separated string. An own Skill keeps only the
// description, metadata and body of the file.
//
// An AI Agent refers to library Skills with skillRefs, each {name, scope}:
// scope is environment (a Skill, the default) or organization (a Skill of the
// organization's library, managed in the dashboard and never declared). The
// list is kept in manifest order and, like skills, replaced as a whole when
// present and left as stored when omitted, so AIAgentSpec.SkillRefs is not
// omitempty and skillRefs: [] unlinks them all. Unlinking never deletes the
// Skill. The agent knows its Skills by name, so a name appears once across
// skillRefs and none is also the name of one of its own Skills; the Control
// Plane checks the same against the stored Skills, and answers 422. It also
// refuses to delete a Skill that an agent refers to.
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
// # GitOps sync
//
// A GitOps runner syncs these custom resources from its cluster to the Control
// Plane (internal/sync, "Control Plane resources"): it checks each with
// Normalize and sends it as ApplyPayload makes it, so the Control Plane reads
// it with the same rules as an apply. A runner syncs them only when it is told
// to (GITOPS_CONTROLPLANE_RESOURCES_ENABLED, off by default). Then every kind in
// the registry is synced, each once the Control Plane advertises its family's
// capability:
// controlplane-resources for Boards, controlplane-resources-ai for MCP
// servers, LLM providers and AI Agents, and controlplane-resources-skills for
// Skills. A resource a GitOps runner owns is read-only to every other writer,
// testkube apply and delete included, and its status.gitOpsOwner names the
// runner.
package declarative
