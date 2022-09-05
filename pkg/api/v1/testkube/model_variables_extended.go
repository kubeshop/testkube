package testkube

import "regexp"

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
			re := regexp.MustCompile(`(` + v.Name + `',\s*')([^\s']*)`)
			output = re.ReplaceAllString(output, "$1*****")
		}
	}
	return output
}
