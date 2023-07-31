package testkube

func VariableTypePtr(stepType VariableType) *VariableType {
	return &stepType
}

var VariableTypeBasic = VariableTypePtr(BASIC_VariableType)
var VariableTypeSecret = VariableTypePtr(SECRET_VariableType)

func VariableTypeString(ptr *VariableType) string {
	return string(*ptr)
}
