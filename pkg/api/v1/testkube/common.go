package testkube

import (
	"fmt"
	"strings"
)

func MapToString(m map[string]string) string {
	labels := []string{}
	for k, v := range m {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(labels, ", ")
}
