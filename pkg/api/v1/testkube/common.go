package testkube

import (
	"fmt"
	"strings"
)

func LabelsToString(labelsMap map[string]string) string {
	labels := []string{}
	for k, v := range labelsMap {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(labels, ", ")
}
