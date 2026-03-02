package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestWebhookCredentialResolution(t *testing.T) {
	tests := []struct {
		name               string
		uri                string
		headers            map[string]string
		payloadTemplate    string
		mockCredentials    map[string]string
		expectedHeaders    map[string]string
		expectedBodyCheck  func(t *testing.T, body []byte)
		expectError        bool
		grpcClientProvider func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient
	}{
		{
			name:            "credential in URI",
			uri:             `https://webhook.example.com/{{credential("api-token")}}`,
			payloadTemplate: `{"event": "{{.Type}}"}`,
			mockCredentials: map[string]string{
				"api-token": "secret-token-123",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBodyCheck: func(t *testing.T, body []byte) {
				var payload map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "start-testworkflow", payload["event"])
			},
			grpcClientProvider: func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient {
				return createMockGRPCClient(ctrl, mockCreds)
			},
		},
		{
			name: "credentials in headers",
			uri:  "https://webhook.example.com/endpoint",
			headers: map[string]string{
				"Authorization": `Bearer {{credential("auth-token")}}`,
				"X-Api-Key":     `{{credential("api-key")}}`,
			},
			payloadTemplate: `{"event": "{{.Type}}"}`,
			mockCredentials: map[string]string{
				"auth-token": "bearer-token-xyz",
				"api-key":    "key-abc-123",
			},
			expectedHeaders: map[string]string{
				"Authorization": "Bearer bearer-token-xyz",
				"X-Api-Key":     "key-abc-123",
				"Content-Type":  "application/json",
			},
			grpcClientProvider: func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient {
				return createMockGRPCClient(ctrl, mockCreds)
			},
		},
		{
			name: "credentials in payload",
			uri:  "https://webhook.example.com/endpoint",
			payloadTemplate: `{
				"event": "{{.Type}}",
				"secret": "{{credential("secret-value")}}",
				"api_key": "{{credential("api-key")}}"
			}`,
			mockCredentials: map[string]string{
				"secret-value": "my-secret-password",
				"api-key":      "key-123",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBodyCheck: func(t *testing.T, body []byte) {
				var payload map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "start-testworkflow", payload["event"])
				assert.Equal(t, "my-secret-password", payload["secret"])
				assert.Equal(t, "key-123", payload["api_key"])
			},
			grpcClientProvider: func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient {
				return createMockGRPCClient(ctrl, mockCreds)
			},
		},
		{
			name:            "backward compatibility - no credentials",
			uri:             "https://webhook.example.com/endpoint",
			payloadTemplate: `{"event": "{{.Type}}", "execution_id": "{{.TestWorkflowExecution.Id}}"}`,
			mockCredentials: map[string]string{},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBodyCheck: func(t *testing.T, body []byte) {
				var payload map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "start-testworkflow", payload["event"])
				assert.Equal(t, "test-exec-123", payload["execution_id"])
			},
			grpcClientProvider: func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient {
				return createMockGRPCClient(ctrl, mockCreds)
			},
		},
		{
			name:               "error when credential client not configured",
			uri:                `https://webhook.example.com/{{credential("api-token")}}`,
			payloadTemplate:    `{"event": "{{.Type}}"}`,
			expectError:        true,
			grpcClientProvider: nil, // No gRPC client
		},
		{
			name:            "error when credential not found",
			uri:             `https://webhook.example.com/{{credential("missing-token")}}`,
			payloadTemplate: `{"event": "{{.Type}}"}`,
			mockCredentials: map[string]string{
				"api-token": "exists",
			},
			expectError: true,
			grpcClientProvider: func(ctrl *gomock.Controller, mockCreds map[string]string) cloud.TestKubeCloudAPIClient {
				return createMockGRPCClient(ctrl, mockCreds)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest *http.Request
			var receivedBody []byte
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedRequest = r
				var err error
				receivedBody, err = io.ReadAll(r.Body)
				require.NoError(t, err)
				w.WriteHeader(http.StatusOK)
			})
			svr := httptest.NewServer(testHandler)
			defer svr.Close()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var grpcClient cloud.TestKubeCloudAPIClient
			if tt.grpcClientProvider != nil {
				grpcClient = tt.grpcClientProvider(ctrl, tt.mockCredentials)
			}

			listenerOpts := []WebhookListenerOption{
				ListenerWithEnvID("test-env-123"),
			}
			if grpcClient != nil {
				listenerOpts = append(listenerOpts, ListenerWithGRPCClient(grpcClient))
			}

			uri := svr.URL
			if tt.expectError {
				uri = tt.uri
			}

			listener := NewWebhookListener(
				"test-webhook",
				uri,
				"",
				[]testkube.EventType{*testkube.EventStartTestWorkflow},
				"",
				tt.payloadTemplate,
				tt.headers,
				false,
				nil,
				nil,
				listenerOpts...,
			)

			event := testkube.Event{
				Type_: testkube.EventStartTestWorkflow,
				TestWorkflowExecution: &testkube.TestWorkflowExecution{
					Id: "test-exec-123",
					Workflow: &testkube.TestWorkflow{
						Name: "test-workflow",
					},
				},
			}

			result := listener.Notify(event)

			if tt.expectError {
				assert.NotEmpty(t, result.Error(), "Expected an error but got none")
			} else {
				assert.Empty(t, result.Error(), "Expected no error but got: %s", result.Error())
				require.NotNil(t, receivedRequest, "Expected HTTP request but got none")

				for key, expectedValue := range tt.expectedHeaders {
					assert.Equal(t, expectedValue, receivedRequest.Header.Get(key), "Header %s mismatch", key)
				}

				if tt.expectedBodyCheck != nil {
					tt.expectedBodyCheck(t, receivedBody)
				}
			}
		})
	}
}

func createMockGRPCClient(ctrl *gomock.Controller, mockCredentials map[string]string) cloud.TestKubeCloudAPIClient {
	mockClient := cloud.NewMockTestKubeCloudAPIClient(ctrl)

	for name, value := range mockCredentials {
		mockClient.EXPECT().
			GetCredential(gomock.Any(), &cloud.CredentialRequest{
				Name:        name,
				ExecutionId: "",
				Source:      "credential",
			}).
			Return(&cloud.CredentialResponse{
				Content: []byte(value),
			}, nil).
			AnyTimes()
	}

	mockClient.EXPECT().
		GetCredential(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *cloud.CredentialRequest, opts ...grpc.CallOption) (*cloud.CredentialResponse, error) {
			if _, ok := mockCredentials[req.Name]; ok {
				return &cloud.CredentialResponse{
					Content: []byte(mockCredentials[req.Name]),
				}, nil
			}
			return nil, assert.AnError
		}).
		AnyTimes()

	return mockClient
}
