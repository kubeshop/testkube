package testkube

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/pkg/secret"
)

type Variables map[string]Variable

func VariablesToMap(v Variables) map[string]string {
	vars := make(map[string]string, len(v))

	for _, v := range v {
		vars[v.Name] = v.Value
	}

	return vars
}

func ObfuscateSecrets(output string, variables Variables, testName string) string {
	// TODO: this is ugly, does anybody have a better idea?
	namespace := "testkube"
	if ns, ok := os.LookupEnv("TESTKUBE_NAMESPACE"); ok {
		namespace = ns
	}
	secretClient, err := secret.NewClient(namespace)
	var secretKeyValues map[string]string
	if err == nil {
		secretKeyValues, _ = secretClient.Get(secret.GetMetadataName(testName))
	}

	for _, v := range variables {
		secretValue := ""
		if *v.Type_ == SECRET_VariableType {
			secretValue = v.Value
		} else if *v.Type_ == SECRET_VariableType && v.SecretRef != nil {
			secretValue = secretKeyValues[v.Value]
		}

		if secretValue != "" {
			output = strings.ReplaceAll(output, `'`+v.Value+`'`, "********")
			output = strings.ReplaceAll(output, `"`+v.Value+`"`, "********")
		}
	}
	return output
}
