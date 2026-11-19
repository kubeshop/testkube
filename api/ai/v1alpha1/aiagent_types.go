package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// AIAgentKind is the kind of an AI Agent.
	AIAgentKind = "AIAgent"
	// AIAgentResource is the plural name of AIAgentKind.
	AIAgentResource = "aiagents"

	// BuiltinMCPServerName is how an AI Agent refers to the environment's
	// built-in Testkube MCP server. No MCPServer may take the name.
	BuiltinMCPServerName = "testkube"

	// SignInNone links an MCP server without signing in.
	SignInNone = "none"
	// SignInSharedKey signs in with one key the whole agent shares, read
	// from credentialReference or, without one, the key set in the
	// dashboard.
	SignInSharedKey = "shared_key"
	// SignInPerPerson signs each person in with their own account.
	SignInPerPerson = "per_person"
	// SignInOBO signs in on behalf of the person through the organization's
	// identity provider, or as the agent's own principal there when nobody
	// is present.
	SignInOBO = "obo"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AIAgent is an AI Agent of an environment. metadata.name is its slug.
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type AIAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIAgentSpec                         `json:"spec,omitempty"`
	Status commonv1.ControlPlaneResourceStatus `json:"status,omitempty"`
}

// AIAgentSpec is what an AI Agent does and what it may use.
type AIAgentSpec struct {
	// DisplayName is the agent's name in the dashboard. It defaults to
	// metadata.name, and is unique within the environment.
	DisplayName string `json:"displayName,omitempty"`
	// Description of the agent.
	Description string `json:"description,omitempty"`
	// Instructions are the agent's system prompt.
	// +kubebuilder:validation:MinLength=1
	Instructions string `json:"instructions"`
	// DefaultModel is the model a chat with the agent starts with. Empty
	// means the platform default.
	DefaultModel string `json:"defaultModel,omitempty"`
	// Labels categorize the agent in the dashboard.
	Labels map[string]string `json:"labels,omitempty"`
	// Private makes the agent visible only to the members linked to it and
	// to the environment's admins. It needs a license that includes RBAC,
	// and has no effect where AI governance is off.
	Private bool `json:"private,omitempty"`
	// MCPServers are the MCP servers the agent uses, each once. The
	// built-in Testkube MCP server is "testkube"; when it is left out, the
	// agent keeps its link to it as it is (the default tools for a new
	// agent).
	MCPServers []AIAgentMCPServer `json:"mcpServers,omitempty"`
	// Access links the agent to roles and resource groups. When it is
	// omitted, the agent's links are left as they are.
	Access *AIAgentAccess `json:"access,omitempty"`
	// Skills are guidance the agent uses, in this order. They belong to the
	// agent: applying a list replaces the whole list, so a Skill left out is
	// deleted, and an empty list removes them all. When skills is omitted,
	// the agent's Skills are left as they are, so applying an older manifest
	// does not delete the ones added in the dashboard.
	// +optional
	Skills []AIAgentSkill `json:"skills"`
	// SkillRefs are Skills of the environment's or the organization's
	// library the agent uses, in this order. They are shared, so a Skill
	// left out is only unlinked. Applying a list replaces the whole list,
	// and skillRefs: [] unlinks them all; when skillRefs is omitted, the
	// agent's links are left as they are. No name may be both a skillRef and
	// one of the agent's own Skills.
	// +optional
	SkillRefs []AIAgentSkillRef `json:"skillRefs"`
}

// AIAgentSkillRef refers to a Skill of a library.
type AIAgentSkillRef struct {
	// Name is the Skill's name: the metadata.name of a Skill, for the
	// environment's library.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[a-z0-9]+(-[a-z0-9]+)*$`
	Name string `json:"name"`
	// Scope is the library: environment (the default) or organization.
	// +kubebuilder:validation:Enum=environment;organization
	Scope string `json:"scope,omitempty"`
}

// AIAgentSkill is guidance an AI Agent uses, following the Agent Skills spec.
// The agent sees its name and description, and reads the content when a task
// needs it, unless it is always included.
type AIAgentSkill struct {
	// Name identifies the Skill within the agent: up to 64 lowercase
	// letters, digits and single hyphens.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[a-z0-9]+(-[a-z0-9]+)*$`
	Name string `json:"name"`
	// Description says what the Skill does and when to use it. Required
	// unless skillFrom is set.
	Description string `json:"description,omitempty"`
	// Content is the guidance itself. Required unless skillFrom is set.
	Content string `json:"content,omitempty"`
	// AlwaysInclude puts the content in every prompt instead of having the
	// agent read it on demand.
	AlwaysInclude bool `json:"alwaysInclude,omitempty"`
	// Metadata is extra Agent Skills spec metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
	// SkillFrom reads the description, metadata and content from a SKILL.md
	// kept in a ConfigMap, whose name must be the Skill's name. Only a GitOps
	// runner reads it, from its own namespace; testkube apply takes the Skill
	// inline.
	SkillFrom *SkillSource `json:"skillFrom,omitempty"`
}

// AIAgentMCPServer is an AI Agent's link to an MCP server.
type AIAgentMCPServer struct {
	// Name is the MCPServer's metadata.name, or "testkube" for the built-in
	// server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// SignIn is how the agent signs in to the server: none, shared_key,
	// per_person or obo (through the organization's identity provider).
	// Empty keeps the behavior of a link made before the choice existed.
	// +kubebuilder:validation:Enum=none;shared_key;per_person;obo
	SignIn string `json:"signIn,omitempty"`
	// CredentialReference names the Environment Credential holding the
	// headers to sign in with, as secret://<name>. It is only valid with
	// shared_key; a shared_key link without one keeps the key set in the
	// dashboard.
	CredentialReference string `json:"credentialReference,omitempty"`
	// Tools configures the server's tools by name. A tool left out is
	// disabled.
	Tools map[string]AIAgentMCPTool `json:"tools,omitempty"`
}

// AIAgentMCPTool configures one tool of an MCP server for an agent.
type AIAgentMCPTool struct {
	// Enabled lets the agent call the tool.
	Enabled bool `json:"enabled,omitempty"`
	// AutoApproved runs the tool without asking for approval first.
	AutoApproved bool `json:"autoApproved,omitempty"`
}

// AIAgentAccess is what an AI Agent is linked to. Linking needs a license
// that includes RBAC.
type AIAgentAccess struct {
	// Roles linked to the agent: environment roles, of read, run, write and
	// admin.
	Roles []string `json:"roles,omitempty"`
	// ResourceGroups linked to the agent, by slug.
	ResourceGroups []string `json:"resourceGroups,omitempty"`
}

//+kubebuilder:object:root=true

// AIAgentList contains a list of AIAgent.
type AIAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIAgent `json:"items"`
}
