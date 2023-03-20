package runner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	kubepug "github.com/rikatz/kubepug/pkg/results"
)

func TestResultParser(t *testing.T) {
	outputDeprecatedAPIs := `[
		{
		  "Description": "ComponentStatus holds the cluster validation info. Deprecated: This API is deprecated in v1.19+",
		  "Group": "",
		  "Kind": "ComponentStatus",
		  "Version": "v1",
		  "Name": "",
		  "Deprecated": true,
		  "Items": [
			{
			  "Scope": "GLOBAL",
			  "ObjectName": "scheduler",
			  "Namespace": ""
			},
			{
			  "Scope": "GLOBAL",
			  "ObjectName": "etcd-0",
			  "Namespace": ""
			},
			{
			  "Scope": "GLOBAL",
			  "ObjectName": "etcd-1",
			  "Namespace": ""
			},
			{
			  "Scope": "GLOBAL",
			  "ObjectName": "controller-manager",
			  "Namespace": ""
			}
		  ]
		}
	  ]`
	outputDeletedAPIs := `[
		{
		  "Group": "extensions",
		  "Kind": "Ingress",
		  "Version": "v1beta1",
		  "Name": "ingresses",
		  "Deleted": true,
		  "Items": [
			{
			  "Scope": "OBJECT",
			  "ObjectName": "cli-testkube-api-server-testkube",
			  "Namespace": "testkube"
			},
			{
			  "Scope": "OBJECT",
			  "ObjectName": "oauth2-proxy",
			  "Namespace": "testkube"
			},
			{
			  "Scope": "OBJECT",
			  "ObjectName": "testapi",
			  "Namespace": "testkube"
			},
			{
			  "Scope": "OBJECT",
			  "ObjectName": "testdash",
			  "Namespace": "testkube"
			},
			{
			  "Scope": "OBJECT",
			  "ObjectName": "testkube-dashboard-testkube",
			  "Namespace": "testkube"
			},
			{
			  "Scope": "OBJECT",
			  "ObjectName": "ui-testkube-api-server-testkube",
			  "Namespace": "testkube"
			}
		  ]
		},
		{
		  "Group": "policy",
		  "Kind": "PodSecurityPolicy",
		  "Version": "v1beta1",
		  "Name": "podsecuritypolicies",
		  "Deleted": true,
		  "Items": [
			{
			  "Scope": "GLOBAL",
			  "ObjectName": "gce.gke-metrics-agent",
			  "Namespace": ""
			}
		  ]
		}
	  ]`

	expectedDeprecatedAPIs := []kubepug.DeprecatedAPI{
		{
			Description: "ComponentStatus holds the cluster validation info. Deprecated: This API is deprecated in v1.19+",
			Kind:        "ComponentStatus",
			Version:     "v1",
			Deprecated:  true,
			Items: []kubepug.Item{
				{
					Scope:      "GLOBAL",
					ObjectName: "scheduler",
				},
				{
					Scope:      "GLOBAL",
					ObjectName: "etcd-0",
				},
				{
					Scope:      "GLOBAL",
					ObjectName: "etcd-1",
				},
				{
					Scope:      "GLOBAL",
					ObjectName: "controller-manager",
				},
			},
		},
	}

	expectedDeletedAPIs := []kubepug.DeletedAPI{
		{
			Group:   "extensions",
			Kind:    "Ingress",
			Version: "v1beta1",
			Name:    "ingresses",
			Deleted: true,
			Items: []kubepug.Item{
				{
					Scope:      "OBJECT",
					ObjectName: "cli-testkube-api-server-testkube",
					Namespace:  "testkube",
				},
				{
					Scope:      "OBJECT",
					ObjectName: "oauth2-proxy",
					Namespace:  "testkube",
				},
				{
					Scope:      "OBJECT",
					ObjectName: "testapi",
					Namespace:  "testkube",
				},
				{
					Scope:      "OBJECT",
					ObjectName: "testdash",
					Namespace:  "testkube",
				},
				{
					Scope:      "OBJECT",
					ObjectName: "testkube-dashboard-testkube",
					Namespace:  "testkube",
				},
				{
					Scope:      "OBJECT",
					ObjectName: "ui-testkube-api-server-testkube",
					Namespace:  "testkube",
				},
			},
		},
		{
			Group:   "policy",
			Kind:    "PodSecurityPolicy",
			Version: "v1beta1",
			Name:    "podsecuritypolicies",
			Deleted: true,
			Items: []kubepug.Item{
				{
					Scope:      "GLOBAL",
					ObjectName: "gce.gke-metrics-agent",
					Namespace:  "",
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
				"DeprecatedAPIs": %s,
				"DeletedAPIs": null
				}`, outputDeprecatedAPIs),
			wantErr: false,
			result: kubepug.Result{
				DeprecatedAPIs: expectedDeprecatedAPIs,
			},
		},
		{
			name: "GetResult should return populated DeletedAPIs when there's a DeletedAPIs finding",
			kubepugOutput: fmt.Sprintf(`{
				"DeprecatedAPIs": null,
				"DeletedAPIs": %s
			  }
			`, outputDeletedAPIs),
			wantErr: false,
			result: kubepug.Result{
				DeletedAPIs: expectedDeletedAPIs,
			},
		},
		{
			name: "GetResult should return populated DeprecatedAPIs and DeletedAPIs when there's both finding",
			kubepugOutput: fmt.Sprintf(`{
				"DeprecatedAPIs": %s,
				"DeletedAPIs": %s
			  }`, outputDeprecatedAPIs, outputDeletedAPIs),
			wantErr: false,
			result: kubepug.Result{
				DeprecatedAPIs: expectedDeprecatedAPIs,
				DeletedAPIs:    expectedDeletedAPIs,
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
