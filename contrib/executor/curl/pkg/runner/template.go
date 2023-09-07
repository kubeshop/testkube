package runner

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/utils"
)

// ResolveTemplates fills the string array with the values if they are templated
func ResolveTemplates(stringsToResolve []string, params map[string]string) error {
	for i := range stringsToResolve {
		finalCommandPart, err := ResolveTemplate(stringsToResolve[i], params)
		stringsToResolve[i] = finalCommandPart
		if err != nil {
			return err
		}
	}

	return nil
}

// ResolveTemplate fills a string with the values if they are templated
func ResolveTemplate(stringToResolve string, params map[string]string) (string, error) {

	ut, err := utils.NewTemplate("cmd").Parse(stringToResolve)

	if err != nil {
		return "", err
	}
	writer := new(strings.Builder)
	err = ut.Execute(writer, params)

	if err != nil {
		return "", err
	}
	return writer.String(), nil
}
