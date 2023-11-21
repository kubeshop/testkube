package runner

import (
	"fmt"
	"testing"

	"github.com/kubepug/kubepug/pkg/apis/v1alpha1"
	kubepug "github.com/kubepug/kubepug/pkg/results"
	"github.com/stretchr/testify/assert"
)

func TestResultParser(t *testing.T) {
	output := `
	[
		{
			"group":"apps",
			"kind":"Deployment",
			"version":"v1beta2",
			"replacement":
				{
					"group":"apps",
					"version":"v1",
					"kind":"Deployment"
				},
			"k8sversion":"1.16",
			"description":"DEPRECATED - This group version of Deployment is deprecated by apps/v1/Deployment. See the release notes for\nmore information.\nDeployment enables declarative updates for Pods and ReplicaSets.",
			"deleted_items":[
				{
					"scope":"OBJECT",
					"objectname":"testkube-dashboard",
					"namespace":"testkube",
					"location":"deployment.yaml"
				}
			]
		}
	]`

	result := []kubepug.ResultItem{
		{
			Description: "DEPRECATED - This group version of Deployment is deprecated by apps/v1/Deployment. See the release notes for\nmore information.\nDeployment enables declarative updates for Pods and ReplicaSets.",
			Group:       "apps",
			Kind:        "Deployment",
			Version:     "v1beta2",
			Replacement: &v1alpha1.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			K8sVersion: "1.16",
			Items: []kubepug.Item{
				{
					Scope:      "OBJECT",
					ObjectName: "testkube-dashboard",
					Namespace:  "testkube",
					Location:   "deployment.yaml",
				},
			},
		},
	}

	tests := []struct {
		name          string
		kubepugOutput string
		wantErr       bool
		result        kubepug.Result
	}{
		{
			name:          "GetResults should return empty result when there is no JSON output",
			kubepugOutput: `{"DeprecatedAPIs":null,"DeletedAPIs":null}`,
			wantErr:       false,
			result: kubepug.Result{
				DeprecatedAPIs: nil,
				DeletedAPIs:    nil,
			},
		},
		{
			name:          "GetResult should return error for invalid JSON",
			kubepugOutput: `invalid JSON`,
			wantErr:       true,
			result:        kubepug.Result{},
		},
		{
			name: "GetResult should return populated DeprecatedAPIs when there's a DeprecatedAPI finding",
			kubepugOutput: fmt.Sprintf(`{
				"deprecated_apis": %s,
				"deleted_apis": null
				}`, output),
			wantErr: false,
			result: kubepug.Result{
				DeprecatedAPIs: result,
			},
		},
		{
			name: "GetResult should return populated DeletedAPIs when there's a DeletedAPIs finding",
			kubepugOutput: fmt.Sprintf(`{
				"deprecated_apis": null,
				"deleted_apis": %s
			  }
			`, output),
			wantErr: false,
			result: kubepug.Result{
				DeletedAPIs: result,
			},
		},
		{
			name: "GetResult should return populated DeprecatedAPIs and DeletedAPIs when there's both finding",
			kubepugOutput: fmt.Sprintf(`{
				"deprecated_apis": %s,
				"deleted_apis": %s
			  }`, output, output),
			wantErr: false,
			result: kubepug.Result{
				DeprecatedAPIs: result,
				DeletedAPIs:    result,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output := tc.kubepugOutput
			result, err := GetResult(output)
			if !tc.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.result, result)
		})
	}
}
