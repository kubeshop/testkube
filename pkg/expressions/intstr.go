package expressions

// These functions are to avoid bundling k8s when only expressions are used.
// It's to be able to work with intstr.IntOrString, without accessing its struct.

type IntOrStringLike interface {
	OpenAPISchemaType() []string
	OpenAPISchemaFormat() string
	OpenAPIV3OneOfTypes() []string
}

func IsIntOrStringType(v any) bool {
	vv, ok := v.(IntOrStringLike)
	return ok && vv.OpenAPISchemaFormat() == "int-or-string"
}
