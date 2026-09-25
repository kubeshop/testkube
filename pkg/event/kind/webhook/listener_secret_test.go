package webhook

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/secret"
)

// A cache serving the wrong value fails silently here, since the webhook still delivers, so the
// value is asserted on the real path rather than on the cache in isolation.
func TestWebhookListener_NotifyWithSecretConfig(t *testing.T) {
	const (
		namespace   = "testkube"
		secretName  = "webhook-token"
		secretKey   = "token"
		secretValue = "s3cr3t-value"
	)

	tests := []struct {
		name string
		ttl  time.Duration
		// wantSecretGets is how often the Kubernetes API is read across two notifications.
		// Without a cache this is not one read per notification: the config map is rebuilt for
		// every templated field, so each notification resolves the secret once for the uri and
		// once for the payload, and once more for every header.
		wantSecretGets int32
	}{
		{
			name:           "a cached client resolves the secret once for repeated notifications",
			ttl:            30 * time.Second,
			wantSecretGets: 1,
		},
		{
			name:           "a disabled cache resolves the secret for every templated field",
			ttl:            0,
			wantSecretGets: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var delivered atomic.Value
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				delivered.Store(string(body))
			}))
			defer server.Close()

			var secretGets atomic.Int32
			clientset := fake.NewClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: secretName},
				Data:       map[string][]byte{secretKey: []byte(secretValue)},
			})
			clientset.PrependReactor("get", "secrets", func(ktesting.Action) (bool, runtime.Object, error) {
				secretGets.Add(1)
				return false, nil, nil
			})

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
			mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", gomock.Any(), gomock.Any()).AnyTimes()

			secretClient := secret.NewCachedClient(secret.NewClientFor(clientset, namespace), namespace, tt.ttl)
			listener := NewWebhookListener("l1", server.URL, "", testEventTypes, "", "{{ .Config.token }}", nil, false,
				map[string]executorv1.WebhookConfigValue{
					secretKey: {Secret: &executorv1.SecretRef{Name: secretName, Key: secretKey}},
				},
				nil,
				listenerWithMetrics(v1.NewMetrics()),
				listenerWithWebhookResultsRepository(mockWebhookRepository),
				listenerWithSecretClient(secretClient))

			for range 2 {
				result := listener.Notify(testkube.Event{
					Type_:                 testkube.EventStartTestWorkflow,
					TestWorkflowExecution: exampleExecution(),
				})
				require.Equal(t, "", result.Error())
				assert.Equal(t, secretValue, delivered.Load(), "the webhook must receive the secret value")
			}

			assert.Equal(t, tt.wantSecretGets, secretGets.Load())
		})
	}
}
