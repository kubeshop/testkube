package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

const (
	// BoardKind is the kind of an Insights board.
	BoardKind = "Board"
	// BoardResource is the plural name of BoardKind.
	BoardResource = "boards"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Board is an Insights board. Boards are organization-scoped in the Control
// Plane, and metadata.name is the board's slug.
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Board struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BoardSpec                           `json:"spec,omitempty"`
	Status commonv1.ControlPlaneResourceStatus `json:"status,omitempty"`
}

// BoardSpec is the content of a board.
type BoardSpec struct {
	// DisplayName is the board's title. It defaults to metadata.name.
	DisplayName string `json:"displayName,omitempty"`
	// Description of the board.
	Description string `json:"description,omitempty"`
	// Shared makes the board visible to the whole organization. It defaults
	// to true, and a board synced by a GitOps agent is always shared.
	Shared *bool `json:"shared,omitempty"`
	// Layout places the reports on rows. When it is omitted every report
	// gets a row of its own, in the order of reports; when it is set it must
	// place every report exactly once.
	Layout *BoardLayout `json:"layout,omitempty"`
	// Reports shown on the board.
	Reports []BoardReport `json:"reports,omitempty"`
}

// BoardLayout places a board's reports on rows.
type BoardLayout struct {
	Rows []BoardLayoutRow `json:"rows"`
}

// BoardLayoutRow is one row of a board layout.
type BoardLayoutRow struct {
	// Cells lists the IDs of the reports in the row, left to right.
	// +kubebuilder:validation:MinItems=1
	Cells []string `json:"cells"`
}

// BoardReport is one report on a board.
type BoardReport struct {
	// ID identifies the report within the board, and is what the layout
	// refers to.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`
	// Kind of the report.
	// +kubebuilder:validation:Enum=time-series;executions;pass-fail;workflows
	Kind string `json:"kind"`
	// Name is the report's title.
	Name string `json:"name,omitempty"`
	// Description of the report.
	Description string `json:"description,omitempty"`
	// Params configure the report, in the dashboard's format (filter,
	// duration, measure, ...). The keys each kind understands are validated
	// when the board is applied.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Params *apiextensionsv1.JSON `json:"params,omitempty"`
}

//+kubebuilder:object:root=true

// BoardList contains a list of Board.
type BoardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Board `json:"items"`
}

