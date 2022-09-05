package testkube

import (
	"strings"
)

type Variables map[string]Variable

func VariablesToMap(v Variables) map[string]string {
	vars := make(map[string]string, len(v))

	for _, v := range v {
		vars[v.Name] = v.Value
	}

	return vars
}

func ObfuscateSecrets(output string, variables Variables) string {
	for _, v := range variables {
		if v.Type_ == VariableTypeSecret {
			output = strings.ReplaceAll(output, `'`+v.Value+`'`, "********")
		}
	}
	return output
}
