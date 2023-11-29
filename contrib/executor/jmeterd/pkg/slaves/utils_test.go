package slaves

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestGetSlavesCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   map[string]testkube.Variable
		want    int
		wantErr bool
	}{
		{
			name:    "Empty Value",
			want:    defaultSlavesCount,
			wantErr: false,
		},
		{
			name:    "Valid Value",
			input:   map[string]testkube.Variable{"SLAVES_COUNT": {Value: "10"}},
			want:    10,
			wantErr: false,
		},
		{
			name:    "Invalid Value",
			input:   map[string]testkube.Variable{"SLAVES_COUNT": {Value: "abc"}},
			want:    0,
			wantErr: true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetSlavesCount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSlavesCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateAndGetSlavePodName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName          string
		executionId       string
		currentSlaveCount int
		expectedOutput    string
	}{
		{
			testName:          "aVeryLongTestNameThatExceedsTheLimitWhenConcatenated",
			executionId:       "exec123",
			currentSlaveCount: 5,
			expectedOutput:    "aVeryLongTestNameTha-slave-5-exec123",
		},
		{
			testName:          "shortName",
			executionId:       "exec123",
			currentSlaveCount: 5,
			expectedOutput:    "shortName-slave-5-exec123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			actualOutput := validateAndGetSlavePodName(tt.testName, tt.executionId, tt.currentSlaveCount)
			if actualOutput != tt.expectedOutput {
				t.Errorf("expected %v, got %v", tt.expectedOutput, actualOutput)
			}
		})
	}
}

func TestIsPodReady(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("poll until pod is ready", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
				PodIP: "192.168.1.1",
			},
		}

		_, err := clientset.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("error injecting pod add: %v", err)
		}

		conditionFunc := isPodReady(clientset, "test-pod", "default")

		// Use PollImmediate to repeatedly evaluate condition
		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*5, time.Second*3, true, conditionFunc)
		if err != nil {
			t.Fatalf("error waiting for pod to be ready: %v", err)
		}
	})

	t.Run("poll times out", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodInitialized,
						Status: v1.ConditionFalse,
					},
				},
				PodIP: "192.168.1.1",
			},
		}

		_, err := clientset.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("error injecting pod add: %v", err)
		}

		conditionFunc := isPodReady(clientset, "test-pod", "default")

		// Use PollImmediate to repeatedly evaluate condition
		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*50, time.Millisecond*160, true, conditionFunc)
		assert.ErrorContains(t, err, "context deadline exceeded")
	})

}
