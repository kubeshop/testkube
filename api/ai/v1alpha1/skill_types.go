package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// SkillKind is the kind of a Skill of an environment's library.
	SkillKind = "Skill"
	// SkillResource is the plural name of SkillKind.
	SkillResource = "skills"

	// SkillScopeEnvironment refers to a Skill of the environment's library,
	// declared as a Skill. It is the default scope of a skillRef.
	SkillScopeEnvironment = "environment"
	// SkillScopeOrganization refers to a Skill of the organization's library,
	// which is managed in the dashboard and never declared as a resource.
	SkillScopeOrganization = "organization"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Skill is a Skill of an environment's library, following the Agent Skills
// spec (https://agentskills.io/specification). metadata.name is the Skill's
// name, and spec follows the SKILL.md frontmatter key for key, with content as
// its Markdown body. AI Agents use it through skillRefs.
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Skill struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SkillSpec                           `json:"spec,omitempty"`
	Status commonv1.ControlPlaneResourceStatus `json:"status,omitempty"`
}

// SkillSpec is a Skill, written inline or read from a SKILL.md with
// skillFrom. With skillFrom, the SKILL.md gives every field but testkube.
type SkillSpec struct {
	// Description says what the Skill does and when to use it: up to 1024
	// characters. Required unless skillFrom is set.
	Description string `json:"description,omitempty"`
	// License is the Skill's license name, or a reference to a license file.
	License string `json:"license,omitempty"`
	// Compatibility says what environment the Skill needs: up to 500
	// characters.
	Compatibility string `json:"compatibility,omitempty"`
	// Metadata is extra Agent Skills spec metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
	// AllowedTools are the tools the Skill may run without asking, space
	// separated, as in a SKILL.md.
	AllowedTools string `json:"allowed-tools,omitempty"`
	// Content is the guidance itself, the Markdown body of a SKILL.md: up to
	// 32000 characters. Required unless skillFrom is set.
	Content string `json:"content,omitempty"`
	// SkillFrom reads the whole Skill, frontmatter and body, from a SKILL.md
	// kept in a ConfigMap, instead of the fields above. Only a GitOps runner
	// reads it, from its own namespace; testkube apply takes the Skill inline.
	SkillFrom *SkillSource `json:"skillFrom,omitempty"`
	// Testkube holds the settings that are Testkube's own rather than the
	// Agent Skills spec's.
	Testkube *SkillTestkubeSettings `json:"testkube,omitempty"`
}

// SkillSource is where a Skill is read from.
type SkillSource struct {
	// ConfigMapKeyRef selects the key of a ConfigMap, in the namespace of the
	// resource, that holds a SKILL.md.
	ConfigMapKeyRef *ConfigMapKeyReference `json:"configMapKeyRef,omitempty"`
}

// ConfigMapKeyReference selects a key of a ConfigMap.
type ConfigMapKeyReference struct {
	// Name of the ConfigMap.
	Name string `json:"name"`
	// Key holding the SKILL.md, such as SKILL.md.
	Key string `json:"key"`
}

// SkillTestkubeSettings are a Skill's Testkube settings.
type SkillTestkubeSettings struct {
	// AlwaysInclude puts the content in every prompt instead of having the
	// agent read it on demand.
	AlwaysInclude bool `json:"alwaysInclude,omitempty"`
	// AddToNewTestBundles adds the Skill to every Test Bundle created after it.
	AddToNewTestBundles bool `json:"addToNewTestBundles,omitempty"`
}

//+kubebuilder:object:root=true

// SkillList contains a list of Skill.
type SkillList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Skill `json:"items"`
}
