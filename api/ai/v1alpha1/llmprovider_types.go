package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// LLMProviderKind is the kind of an LLM provider.
	LLMProviderKind = "LLMProvider"
	// LLMProviderResource is the plural name of LLMProviderKind.
	LLMProviderResource = "llmproviders"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LLMProvider is a model provider an environment brings its own key for.
// metadata.name is its slug. The key is never part of the manifest: it is
// named by credentialReference and stays in the Control Plane.
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type LLMProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMProviderSpec                     `json:"spec,omitempty"`
	Status commonv1.ControlPlaneResourceStatus `json:"status,omitempty"`
}

// LLMProviderSpec is how to reach a provider and which of its models to offer.
type LLMProviderSpec struct {
	// DisplayName is the provider's name in the dashboard. It defaults to
	// metadata.name.
	DisplayName string `json:"displayName,omitempty"`
	// Provider is the kind of provider, such as openai, anthropic or azure.
	// +kubebuilder:validation:MinLength=1
	Provider string `json:"provider"`
	// BaseURL overrides the provider's default endpoint.
	BaseURL string `json:"baseUrl,omitempty"`
	// CredentialReference names the Environment Credential holding the key:
	// secret://<name> (or a bare name) for a credential, vault://<path> for
	// a Vault secret.
	CredentialReference string `json:"credentialReference,omitempty"`
	// ExtraHeaders are sent with every request to the provider. They must
	// not carry a credential: a header such as Authorization or X-Api-Key is
	// refused, and belongs in the credential.
	ExtraHeaders map[string]string `json:"extraHeaders,omitempty"`
	// QueryParams are added to every request to the provider, under the same
	// rule as ExtraHeaders.
	QueryParams map[string]string `json:"queryParams,omitempty"`
	// Models offered from the provider.
	Models []LLMProviderModel `json:"models,omitempty"`
}

// LLMProviderModel is one model offered from a provider.
type LLMProviderModel struct {
	// Name is the model's name at the provider, unique within the provider.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Label is the model's name in the dashboard.
	Label string `json:"label,omitempty"`
	// Type of the model, such as chat or embedding.
	Type string `json:"type,omitempty"`
	// URL overrides the provider's endpoint for this model.
	URL string `json:"url,omitempty"`
	// ContextWindowTokens is the size of the model's context window.
	// +kubebuilder:validation:Minimum=0
	ContextWindowTokens int `json:"contextWindowTokens,omitempty"`
	// ExtraHeaders are sent with requests for this model, under the
	// provider's rule.
	ExtraHeaders map[string]string `json:"extraHeaders,omitempty"`
	// QueryParams are added to requests for this model, under the
	// provider's rule.
	QueryParams map[string]string `json:"queryParams,omitempty"`
}

//+kubebuilder:object:root=true

// LLMProviderList contains a list of LLMProvider.
type LLMProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMProvider `json:"items"`
}
