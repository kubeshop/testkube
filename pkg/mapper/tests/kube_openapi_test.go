package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapTestCRToAPI(t *testing.T) {
	test := testsv3.Test{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Test",
			APIVersion: "tests.testkube.io/v3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-1",
			Namespace:         "testkube",
			CreationTimestamp: metav1.NewTime(time.Time{}),
		},
		Spec: testsv3.TestSpec{
			Type_: "curl/test",
			Name:  "test-1",
			Content: &testsv3.TestContent{
				Type_: "string",
				Data:  "123",
			},
			ExecutionRequest: &testsv3.ExecutionRequest{
				Name:   "",
				Number: 1,
				ExecutionLabels: map[string]string{
					"": "",
				},
				Args:    []string{"-v", "-X", "GET", "https://testkube.kubeshop.io"},
				Command: []string{"curl"},
				Image:   "curlimages/curl:7.85.0",
				ImagePullSecrets: []v1.LocalObjectReference{
					{Name: "secret"},
				},
				Envs: map[string]string{
					"TESTKUBE": "1",
				},
				Sync: false,
				EnvConfigMaps: []testsv3.EnvReference{
					{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "configmap",
						},
						Mount:          true,
						MountPath:      "/data",
						MapToVariables: false,
					},
				},
				EnvSecrets: []testsv3.EnvReference{
					{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "secret",
						},
						Mount:          false,
						MountPath:      "",
						MapToVariables: true,
					},
				},
			},
		},
	}
	got := MapTestCRToAPI(test)

	want := testkube.Test{
		Name:      "test-1",
		Namespace: "testkube",
		Type_:     "curl/test",
		Content: &testkube.TestContent{
			Type_: "string",
			Data:  "123",
		},
		Created: time.Time{},
		ExecutionRequest: &testkube.ExecutionRequest{
			Name:   "",
			Number: 1,
			ExecutionLabels: map[string]string{
				"": "",
			},
			Namespace:     "",
			Variables:     map[string]testkube.Variable{},
			VariablesFile: "",
			Args:          []string{"-v", "-X", "GET", "https://testkube.kubeshop.io"},
			ArgsMode:      "",
			Command:       []string{"curl"},
			Image:         "curlimages/curl:7.85.0",
			ImagePullSecrets: []testkube.LocalObjectReference{
				{Name: "secret"},
			},
			Envs: map[string]string{
				"TESTKUBE": "1",
			},
			Sync: false,
			EnvConfigMaps: []testkube.EnvReference{
				{
					Reference: &testkube.LocalObjectReference{
						Name: "configmap",
					},
					Mount:          true,
					MountPath:      "/data",
					MapToVariables: false,
				},
			},
			EnvSecrets: []testkube.EnvReference{
				{
					Reference: &testkube.LocalObjectReference{
						Name: "secret",
					},
					Mount:          false,
					MountPath:      "",
					MapToVariables: true,
				},
			},
		},
	}
	assert.Equal(t, want, got)
}
