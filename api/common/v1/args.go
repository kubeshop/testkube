package v1

// ArgsModeType defines args mode type
// +kubebuilder:validation:Enum=append;override;replace
type ArgsModeType string

const (
	// ArgsModeTypeAppend for append args mode
	ArgsModeTypeAppend ArgsModeType = "append"
	// ArgsModeTypeOverride for override args mode
	ArgsModeTypeOverride ArgsModeType = "override"
	// ArgsModeTypeReplace for replace args mode
	ArgsModeTypeReplace ArgsModeType = "replace"
)
