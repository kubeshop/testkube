package testworkflowprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/fake"
)

func TestJobCreateError(t *testing.T) {
	job := schema.GroupKind{Group: "batch", Kind: "Job"}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a cause that every container repeats appears once, without the job ID",
			err: apierrors.NewInvalid(job, "6ac00fe963024bd198b08985", field.ErrorList{
				field.NotFound(field.NewPath("spec", "template", "spec", "containers").Index(0).Child("volumeMounts").Index(4).Child("name"), "missing-volume"),
				field.NotFound(field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("volumeMounts").Index(4).Child("name"), "missing-volume"),
			}),
			want: `the job is invalid: volumeMounts[4].name: Not found: "missing-volume"`,
		},
		{
			name: "different causes all appear",
			err: apierrors.NewInvalid(job, "6ac00fe963024bd198b08985", field.ErrorList{
				field.Required(field.NewPath("spec", "template", "spec", "volumes").Index(0).Child("name"), ""),
				field.Invalid(field.NewPath("spec", "template", "spec", "containers").Index(1).Child("image"), "", "must not be empty"),
			}),
			want: `the job is invalid: volumes[0].name: Required value; image: Invalid value: "": must not be empty`,
		},
		{
			name: "a refusal without causes keeps the words of Kubernetes",
			err:  apierrors.NewForbidden(schema.GroupResource{Group: "batch", Resource: "jobs"}, "6ac00fe963024bd198b08985", errors.New("exceeded quota")),
			want: `jobs.batch "6ac00fe963024bd198b08985" is forbidden: exceeded quota`,
		},
		{
			name: "an error that did not come from Kubernetes keeps its text",
			err:  errors.New("connection refused"),
			want: "connection refused",
		},
		{
			name: "no error stays no error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := jobCreateError(tt.err)
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestBundle_Deploy_Namespace(t *testing.T) {
	tests := []struct {
		name          string
		jobNamespace  string
		wantNamespace string
	}{
		{name: "the job without a namespace runs in the given namespace", jobNamespace: "", wantNamespace: "src"},
		{name: "the namespace of the job wins over the given namespace", jobNamespace: "job-ns", wantNamespace: "job-ns"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientSet := fake.NewSimpleClientset()
			b := &Bundle{Job: batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Namespace: tt.jobNamespace}}}

			require.NoError(t, b.Deploy(context.Background(), clientSet, "src"))

			assert.Equal(t, tt.wantNamespace, b.Job.Namespace)
			_, err := clientSet.BatchV1().Jobs(tt.wantNamespace).Get(context.Background(), "exec-1", metav1.GetOptions{})
			assert.NoError(t, err)
		})
	}
}
