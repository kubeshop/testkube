package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	executorsv1 "github.com/kubeshop/testkube/api/executor/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
)

func TestWebhookLoader(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
	mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: defaultWebhookSpec(nil)},
		},
	}, nil).AnyTimes()
	mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(
		mockWebhooksClient,
		WithMetrics(v1.NewMetrics()),
		WithWebhookResultsRepository(mockWebhookRepository),
	)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}

func TestWebhookTemplateLoader(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
	mockWebhookTemplatesClient := executorsclientv1.NewMockWebhookTemplatesInterface(mockCtrl)
	mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{WebhookTemplateRef: &executorsv1.WebhookTemplateRef{Name: "name"}}},
		},
	}, nil).AnyTimes()
	mockWebhookTemplatesClient.EXPECT().Get("name").Return(&executorsv1.WebhookTemplate{
		Spec: executorsv1.WebhookTemplateSpec{
			Uri: "http://localhost:3333", Events: []executorsv1.EventType{"start-test"}, PayloadObjectField: "text", PayloadTemplate: "{{ .Id }}", Headers: map[string]string{"Content-Type": "application/xml"},
		},
	}, nil).AnyTimes()

	mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(
		mockWebhooksClient,
		WithWebhookTemplateClient(mockWebhookTemplatesClient),
		WithMetrics(v1.NewMetrics()),
		WithWebhookResultsRepository(mockWebhookRepository),
	)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}

func TestWebhookLoader_TargetMatching(t *testing.T) {
	testCases := []struct {
		name           string
		target         *commonv1.Target
		agentID        string
		agentName      string
		agentLabels    map[string]string
		expectedLoaded int
	}{
		{
			name:           "no target keeps backward compatibility",
			target:         nil,
			expectedLoaded: 1,
		},
		{
			name:           "empty target keeps backward compatibility",
			target:         &commonv1.Target{},
			expectedLoaded: 1,
		},
		{
			name: "label match includes webhook",
			target: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"accounting"},
				},
			},
			agentLabels:    map[string]string{"application": "accounting"},
			expectedLoaded: 1,
		},
		{
			name: "label mismatch excludes webhook",
			target: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"accounting"},
				},
			},
			agentLabels:    map[string]string{"application": "billing"},
			expectedLoaded: 0,
		},
		{
			name: "id and name match includes webhook",
			target: &commonv1.Target{
				Match: map[string][]string{
					"id":   []string{"agent-1"},
					"name": []string{"runner-us-east"},
				},
			},
			agentID:        "agent-1",
			agentName:      "runner-us-east",
			expectedLoaded: 1,
		},
		{
			name: "not selector excludes webhook",
			target: &commonv1.Target{
				Not: map[string][]string{
					"region": []string{"deprecated"},
				},
			},
			agentLabels:    map[string]string{"region": "deprecated"},
			expectedLoaded: 0,
		},
		{
			name: "not selector does not exclude when value differs",
			target: &commonv1.Target{
				Not: map[string][]string{
					"region": []string{"deprecated"},
				},
			},
			agentLabels:    map[string]string{"region": "us-east"},
			expectedLoaded: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
			mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
				Items: []executorsv1.Webhook{
					{
						Spec: defaultWebhookSpec(tc.target),
					},
				},
			}, nil)

			mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
			webhooksLoader := NewWebhookLoader(
				mockWebhooksClient,
				WithMetrics(v1.NewMetrics()),
				WithWebhookResultsRepository(mockWebhookRepository),
				WithAgentID(tc.agentID),
				WithAgentName(tc.agentName),
				WithAgentLabels(tc.agentLabels),
			)
			listeners, err := webhooksLoader.Load()

			assert.NoError(t, err)
			assert.Len(t, listeners, tc.expectedLoaded)
		})
	}
}

func TestWebhookTemplateLoader_TargetInheritanceAndOverride(t *testing.T) {
	testCases := []struct {
		name           string
		webhookTarget  *commonv1.Target
		templateTarget *commonv1.Target
		agentLabels    map[string]string
		expectedLoaded int
	}{
		{
			name:          "inherits template target when webhook target is missing",
			webhookTarget: nil,
			templateTarget: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"accounting"},
				},
			},
			agentLabels:    map[string]string{"application": "accounting"},
			expectedLoaded: 1,
		},
		{
			name:          "template target can exclude webhook",
			webhookTarget: nil,
			templateTarget: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"accounting"},
				},
			},
			agentLabels:    map[string]string{"application": "billing"},
			expectedLoaded: 0,
		},
		{
			name: "webhook target overrides template target",
			webhookTarget: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"billing"},
				},
			},
			templateTarget: &commonv1.Target{
				Match: map[string][]string{
					"application": []string{"accounting"},
				},
			},
			agentLabels:    map[string]string{"application": "billing"},
			expectedLoaded: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
			mockWebhookTemplatesClient := executorsclientv1.NewMockWebhookTemplatesInterface(mockCtrl)
			mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
				Items: []executorsv1.Webhook{
					{
						Spec: executorsv1.WebhookSpec{
							WebhookTemplateRef: &executorsv1.WebhookTemplateRef{Name: "name"},
							Target:             tc.webhookTarget,
						},
					},
				},
			}, nil)
			mockWebhookTemplatesClient.EXPECT().Get("name").Return(&executorsv1.WebhookTemplate{
				Spec: executorsv1.WebhookTemplateSpec{
					Uri:     "http://localhost:3333",
					Events:  []executorsv1.EventType{"start-test"},
					Target:  tc.templateTarget,
					Headers: map[string]string{"Content-Type": "application/xml"},
				},
			}, nil)

			mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
			webhooksLoader := NewWebhookLoader(
				mockWebhooksClient,
				WithWebhookTemplateClient(mockWebhookTemplatesClient),
				WithMetrics(v1.NewMetrics()),
				WithWebhookResultsRepository(mockWebhookRepository),
				WithAgentLabels(tc.agentLabels),
			)
			listeners, err := webhooksLoader.Load()

			assert.NoError(t, err)
			assert.Len(t, listeners, tc.expectedLoaded)
		})
	}
}

func defaultWebhookSpec(target *commonv1.Target) executorsv1.WebhookSpec {
	return executorsv1.WebhookSpec{
		Uri:                "http://localhost:3333",
		Events:             []executorsv1.EventType{"start-test"},
		PayloadObjectField: "text",
		PayloadTemplate:    "{{ .Id }}",
		Headers:            map[string]string{"Content-Type": "application/xml"},
		Target:             target,
	}
}
