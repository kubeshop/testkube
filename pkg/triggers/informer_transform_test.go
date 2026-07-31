package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	faketestkube "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/fake"
)

func managedFieldsEntries() []metav1.ManagedFieldsEntry {
	return []metav1.ManagedFieldsEntry{{
		Manager:    "kubectl",
		Operation:  metav1.ManagedFieldsOperationApply,
		APIVersion: "v1",
		FieldsType: "FieldsV1",
	}}
}

func TestStripManagedFields(t *testing.T) {
	tests := []struct {
		name         string
		obj          any
		wantStripped bool
	}{
		{
			name: "typed object loses its managed fields",
			obj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Namespace:     "testkube",
				Name:          "pod",
				ManagedFields: managedFieldsEntries(),
			}},
			wantStripped: true,
		},
		{
			name: "unstructured object loses its managed fields",
			obj: &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"namespace": "testkube",
					"name":      "config",
					"managedFields": []any{map[string]any{
						"manager":    "kubectl",
						"operation":  "Apply",
						"apiVersion": "v1",
					}},
				},
			}},
			wantStripped: true,
		},
		{
			name: "tombstone is passed through",
			obj: cache.DeletedFinalStateUnknown{
				Key: "testkube/pod",
				Obj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace:     "testkube",
					Name:          "pod",
					ManagedFields: managedFieldsEntries(),
				}},
			},
		},
		{
			name: "value without object metadata is passed through",
			obj:  "not a kubernetes object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stripManagedFields(tt.obj)
			require.NoError(t, err)
			assert.Equal(t, tt.obj, result)

			if !tt.wantStripped {
				return
			}

			accessor, err := meta.Accessor(result)
			require.NoError(t, err)
			assert.Empty(t, accessor.GetManagedFields())
		})
	}
}

func TestService_runInformers_managedFieldsTransform(t *testing.T) {
	namespace := "testkube"

	tests := []struct {
		name              string
		keepManagedFields bool
		wantKept          bool
	}{
		{
			name:     "informer caches drop managed fields by default",
			wantKept: false,
		},
		{
			name:              "informer caches keep managed fields when configured to",
			keepManagedFields: true,
			wantKept:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Seed the objects at construction: the fake clientset runs its own field
			// management on Create, which would replace the managed fields under test.
			clientset := fake.NewClientset(
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace:     namespace,
					Name:          "pod",
					ManagedFields: managedFieldsEntries(),
				}},
				&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
					Namespace:     namespace,
					Name:          "config-map",
					ManagedFields: managedFieldsEntries(),
				}},
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{
					Namespace:     namespace,
					Name:          "service",
					ManagedFields: managedFieldsEntries(),
				}},
			)
			service := newWatcherTestService(clientset, faketestkube.NewSimpleClientset(), namespace)
			service.keepManagedFields = tt.keepManagedFields

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go service.runWatcher(ctx)

			require.Eventually(t, func() bool {
				service.informersMu.RLock()
				defer service.informersMu.RUnlock()
				return service.informers != nil
			}, time.Second, 10*time.Millisecond)

			service.informersMu.RLock()
			informers := service.informers
			service.informersMu.RUnlock()

			// One case per wiring site, since the transform is installed per informer.
			caches := []struct {
				name  string
				key   string
				store cache.Store
			}{
				{name: "pod", key: namespace + "/pod", store: informers.podInformers[0].Informer().GetStore()},
				{name: "config map", key: namespace + "/config-map", store: informers.configMapInformers[0].Informer().GetStore()},
				{name: "service", key: namespace + "/service", store: informers.serviceInformers[0].Informer().GetStore()},
			}

			for _, c := range caches {
				t.Run(c.name, func(t *testing.T) {
					var cached any
					require.Eventually(t, func() bool {
						item, exists, err := c.store.GetByKey(c.key)
						if err != nil || !exists {
							return false
						}
						cached = item
						return true
					}, 5*time.Second, 10*time.Millisecond)

					accessor, err := meta.Accessor(cached)
					require.NoError(t, err)
					if tt.wantKept {
						assert.Equal(t, managedFieldsEntries(), accessor.GetManagedFields())
						return
					}
					assert.Empty(t, accessor.GetManagedFields())
				})
			}
		})
	}
}
