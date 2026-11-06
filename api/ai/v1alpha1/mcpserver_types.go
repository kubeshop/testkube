package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// MCPServerKind is the kind of an MCP server.
	MCPServerKind = "MCPServer"
	// MCPServerResource is the plural name of MCPServerKind.
	MCPServerResource = "mcpservers"

	// MCPServerTypeHTTP connects to a streamable HTTP endpoint.
	MCPServerTypeHTTP = "http"
	// MCPServerTypeSSE connects to a Server-Sent Events endpoint.
	MCPServerTypeSSE = "sse"
	// MCPServerTypeStdio runs a local command.
	MCPServerTypeStdio = "stdio"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MCPServer is an MCP server that AI Agents of an environment can use.
// metadata.name is its slug, which AI Agents refer to it by. How an agent
// signs in to the server is set on the agent's link to it, not here.
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServerSpec   `json:"spec,omitempty"`
	Status MCPServerStatus `json:"status,omitempty"`
}

// MCPServerSpec is how to reach an MCP server.
type MCPServerSpec struct {
	// DisplayName is the server's name in the dashboard. It defaults to
	// metadata.name, and is unique within the environment.
	DisplayName string `json:"displayName,omitempty"`
	// Description of the server.
	Description string `json:"description,omitempty"`
	// Type of connection. It defaults to http.
	// +kubebuilder:validation:Enum=http;sse;stdio
	Type string `json:"type,omitempty"`
	// URL of the endpoint. The Control Plane requires one for every type,
	// and checks it is a public HTTPS address unless the host is trusted.
	URL string `json:"url,omitempty"`
	// Command to run, required for stdio.
	Command string `json:"command,omitempty"`
	// Args of the command, for stdio.
	Args []string `json:"args,omitempty"`
}

// MCPServerStatus is the Control Plane's view of an MCP server.
type MCPServerStatus struct {
	commonv1.ControlPlaneResourceStatus `json:",inline"`
	// Connection is the result of the last connection: unknown, active or
	// error.
	Connection string `json:"connection,omitempty"`
	// LastError is the error of the last failed connection.
	LastError string `json:"lastError,omitempty"`
	// LastConnectedAt is when the server was last reached.
	LastConnectedAt *metav1.Time `json:"lastConnectedAt,omitempty"`
}

//+kubebuilder:object:root=true

// MCPServerList contains a list of MCPServer.
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServer `json:"items"`
}
