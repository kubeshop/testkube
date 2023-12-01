package options

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	"github.com/kubeshop/testkube/internal/featureflags"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/log"
)

var defaultJobTemplate = ``

func TestNewExecutorJobSpecEmptyArgs(t *testing.T) {
	t.Parallel()

	jobOptions := JobOptions{
		Name:                      "name",
		Namespace:                 "namespace",
		InitImage:                 "kubeshop/testkube-init-executor:0.7.10",
		Image:                     "ubuntu",
		JobTemplate:               defaultJobTemplate,
		ScraperTemplate:           "",
		PvcTemplate:               "",
		JobTemplateExtensions:     "",
		ScraperTemplateExtensions: "",
		PvcTemplateExtensions:     "",
		Command:                   []string{},
		Args:                      []string{},
		Features:                  featureflags.FeatureFlags{},
	}
	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)
}

func TestNewExecutorJobSpecWithArgs(t *testing.T) {
	t.Parallel()

	jobOptions := JobOptions{
		Name:                      "name",
		Namespace:                 "namespace",
		InitImage:                 "kubeshop/testkube-init-executor:0.7.10",
		Image:                     "curl",
		JobTemplate:               defaultJobTemplate,
		ScraperTemplate:           "",
		PvcTemplate:               "",
		JobTemplateExtensions:     "",
		ScraperTemplateExtensions: "",
		PvcTemplateExtensions:     "",
		ImagePullSecrets:          []string{"secret-name"},
		Command:                   []string{"/bin/curl"},
		Args:                      []string{"-v", "https://testkube.kubeshop.io"},
		ActiveDeadlineSeconds:     100,
		Envs:                      map[string]string{"key": "value"},
		Variables:                 map[string]testkube.Variable{"aa": {Name: "aa", Value: "bb", Type_: testkube.VariableTypeBasic}},
		Features:                  featureflags.FeatureFlags{},
	}
	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)

	assert.NoError(t, err)
	assert.NotNil(t, spec)

	wantEnvs := []corev1.EnvVar{
		{Name: "DEBUG", Value: ""},
		{Name: "RUNNER_ENDPOINT", Value: ""},
		{Name: "RUNNER_ACCESSKEYID", Value: ""},
		{Name: "RUNNER_SECRETACCESSKEY", Value: ""},
		{Name: "RUNNER_REGION", Value: ""},
		{Name: "RUNNER_TOKEN", Value: ""},
		{Name: "RUNNER_BUCKET", Value: ""},
		{Name: "RUNNER_SSL", Value: "false"},
		{Name: "RUNNER_SKIP_VERIFY", Value: "false"},
		{Name: "RUNNER_CERT_FILE", Value: ""},
		{Name: "RUNNER_KEY_FILE", Value: ""},
		{Name: "RUNNER_CA_FILE", Value: ""},
		{Name: "RUNNER_SCRAPPERENABLED", Value: "false"},
		{Name: "RUNNER_DATADIR", Value: "/data"},
		{Name: "RUNNER_CDEVENTS_TARGET", Value: ""},
		{Name: "RUNNER_DASHBOARD_URI", Value: ""},
		{Name: "RUNNER_COMPRESSARTIFACTS", Value: "false"},
		{Name: "RUNNER_WORKINGDIR", Value: ""},
		{Name: "RUNNER_EXECUTIONID", Value: "name"},
		{Name: "RUNNER_TESTNAME", Value: ""},
		{Name: "RUNNER_EXECUTIONNUMBER", Value: "0"},
		{Name: "RUNNER_CONTEXTTYPE", Value: ""},
		{Name: "RUNNER_CONTEXTDATA", Value: ""},
		{Name: "RUNNER_APIURI", Value: ""},
		{Name: "RUNNER_CLOUD_MODE", Value: "false"},
		{Name: "RUNNER_CLOUD_API_KEY", Value: ""},
		{Name: "RUNNER_CLOUD_API_URL", Value: ""},
		{Name: "RUNNER_CLOUD_API_TLS_INSECURE", Value: "false"},
		{Name: "RUNNER_CLUSTERID", Value: ""},
		{Name: "CI", Value: "1"},
		{Name: "key", Value: "value"},
		{Name: "aa", Value: "bb"},
	}

	assert.ElementsMatch(t, wantEnvs, spec.Spec.Template.Spec.Containers[0].Env)
}

func TestNewExecutorJobSpecWithoutInitImage(t *testing.T) {
	t.Parallel()

	jobOptions := JobOptions{
		Name:                      "name",
		Namespace:                 "namespace",
		InitImage:                 "",
		Image:                     "ubuntu",
		JobTemplate:               defaultJobTemplate,
		ScraperTemplate:           "",
		PvcTemplate:               "",
		JobTemplateExtensions:     "",
		ScraperTemplateExtensions: "",
		PvcTemplateExtensions:     "",
		Command:                   []string{},
		Args:                      []string{},
		Features:                  featureflags.FeatureFlags{},
	}
	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)
}

func TestNewExecutorJobSpecWithWorkingDirRelative(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)

	jobOptions, _ := NewJobOptions(
		log.DefaultLogger,
		mockTemplatesClient,
		executor.Images{},
		executor.Templates{},
		"",
		"",
		"",
		"",
		testkube.Execution{
			Id:            "name",
			TestName:      "name-test-1",
			TestNamespace: "namespace",
		},
		ExecuteOptions{
			TestSpec: testsv3.TestSpec{
				ExecutionRequest: &testsv3.ExecutionRequest{
					Image: "ubuntu",
				},
				Content: &testsv3.TestContent{
					Repository: &testsv3.Repository{
						WorkingDir: "relative/path",
					},
				},
			},
		},
		"",
		false,
	)

	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)

	assert.Equal(t, repoPath+"/relative/path", spec.Spec.Template.Spec.Containers[0].WorkingDir)
}

func TestNewExecutorJobSpecWithWorkingDirAbsolute(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)

	jobOptions, _ := NewJobOptions(
		log.DefaultLogger,
		mockTemplatesClient,
		executor.Images{},
		executor.Templates{},
		"",
		"",
		"",
		"",
		testkube.Execution{
			Id:            "name",
			TestName:      "name-test-1",
			TestNamespace: "namespace",
		},
		ExecuteOptions{
			TestSpec: testsv3.TestSpec{
				ExecutionRequest: &testsv3.ExecutionRequest{
					Image: "ubuntu",
				},
				Content: &testsv3.TestContent{
					Repository: &testsv3.Repository{
						WorkingDir: "/absolute/path",
					},
				},
			},
		},
		"",
		false,
	)
	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)

	assert.Equal(t, "/absolute/path", spec.Spec.Template.Spec.Containers[0].WorkingDir)
}

func TestNewExecutorJobSpecWithoutWorkingDir(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)

	jobOptions, _ := NewJobOptions(
		log.DefaultLogger,
		mockTemplatesClient,
		executor.Images{},
		executor.Templates{},
		"",
		"",
		"",
		"",
		testkube.Execution{
			Id:            "name",
			TestName:      "name-test-1",
			TestNamespace: "namespace",
		},
		ExecuteOptions{
			Namespace: "namespace",
			TestSpec: testsv3.TestSpec{
				ExecutionRequest: &testsv3.ExecutionRequest{
					Image: "ubuntu",
				},
				Content: &testsv3.TestContent{
					Repository: &testsv3.Repository{},
				},
			},
		},
		"",
		false,
	)
	spec, err := NewExecutorJobSpec(log.DefaultLogger, jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)

	assert.Empty(t, spec.Spec.Template.Spec.Containers[0].WorkingDir)
}
