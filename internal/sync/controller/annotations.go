package controller

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const noGitOpsSyncAnnotation = "testkube.io/no-gitops-sync"

func hasNoGitOpsSyncAnnotation(obj metav1.Object) bool {
	value, ok := obj.GetAnnotations()[noGitOpsSyncAnnotation]
	if !ok {
		return false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}

	return parsed
}
