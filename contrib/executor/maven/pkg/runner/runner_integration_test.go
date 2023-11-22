package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRun_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("run maven wrapper test goal with envs", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-mvnw", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		_, err = NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "someuri",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}

		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN_WRAPPER", Value: "true", Type_: testkube.VariableTypeBasic},
		}

		assert.NoError(t, os.Setenv("TESTKUBE_MAVEN_WRAPPER", "true"))
		// when

		// TODO: fix flaky tests:  TKC-923
		// result, err := runner.Run(ctx, *execution)

		// then
		// assert.NoError(t, err)
		// assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		// assert.Len(t, result.Steps, 1)
	})

	t.Run("run maven project with test task and envs", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-maven", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "someur",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"test", "--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}
		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN", Value: "true", Type_: testkube.VariableTypeBasic},
		}

		assert.NoError(t, os.Setenv("TESTKUBE_MAVEN", "true"))
		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("run maven wrapper test goal with envs", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-mvnw", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}
		execution.Envs = map[string]string{"TESTKUBE_MAVEN_WRAPPER": "true"}

		assert.NoError(t, os.Setenv("TESTKUBE_MAVEN_WRAPPER", "true"))
		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("run maven project with settings.xml", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-maven-settings", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "someuri",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}
		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN", Value: "true", Type_: testkube.VariableTypeBasic},
		}
		settingsContent, err := os.ReadFile(fmt.Sprintf("%s/settings.xml", repoDir))
		assert.NoError(t, err)
		execution.VariablesFile = string(settingsContent)

		assert.NoError(t, os.Setenv("TESTKUBE_MAVEN", "true"))
		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunErrors_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("no RUNNER_DATADIR", func(t *testing.T) {
		t.Parallel()
		// given
		params := envs.Params{DataDir: "/unknown"}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		assert.Error(t, err)
	})

	t.Run("unsupported file-content", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		assert.Error(t, err)
	})

	t.Run("no pom.xml", func(t *testing.T) {
		t.Parallel()

		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Contains(t, result.ErrorMessage, "no")
	})
}

func TestRunMavenProject_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("run maven project with test task and envs", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-maven", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"mvn"}
		execution.Args = []string{"test", "--settings", "<settingsFile>", "<goalName>", "-Duser.home", "<mavenHome>"}
		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN", Value: "true", Type_: testkube.VariableTypeBasic},
		}

		assert.NoError(t, os.Setenv("TESTKUBE_MAVEN", "true"))

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 1)
	})
}
