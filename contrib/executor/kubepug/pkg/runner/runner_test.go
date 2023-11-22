package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRunString_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should return success and empty result on empty string", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(""), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write postman runner test content file")
		}

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "passed", result.Steps[1].Status)
	})

	t.Run("runner should return success and empty result on passing yaml", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    control-plane.alpha.kubernetes.io/leader: '{"holderIdentity":"ingress-nginx-controller-646d5d4d67-7nx7r"}'
  creationTimestamp: "2021-10-07T13:44:37Z"
  name: ingress-controller-leader
  namespace: default
  resourceVersion: "170745168"
  uid: 9bb57467-b5c4-41fe-83a8-9513ae86fbff
`

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(data), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write postman runner test content file")
		}

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}
		execution.Content = testkube.NewStringTestContent("")

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "passed", result.Steps[1].Status)
	})

	t.Run("runner should return failure and list of deprecated APIs result on yaml containing deprecated API", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: v1
kind: Service
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  ports:
    - name: http
      port: 8001
      targetPort: 80
  selector:
    app: testkube-dashboard
---
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testkube-dashboard
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: testkube-dashboard
    spec:
      containers:
        - name: petstore-backend
          image: docker.io/kubeshop/testkube-dashboard:latest
          ports:
            - name: http
              containerPort: 80
          livenessProbe:
            httpGet:
            path: /index.html
            port: 80
            initialDelaySeconds: 15
            timeoutSeconds: 2
            periodSeconds: 5
            failureThreshold: 1
          readinessProbe:
            httpGet:
              path: /index.html
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 5
            failureThreshold: 1
`

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(data), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write postman runner test content file")
		}

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}
		execution.Content = testkube.NewStringTestContent("")

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "failed", result.Steps[1].Status)
	})
}

func TestRunFileURI_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should return success on valid yaml gist file URI", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}
		uri := "https://gist.githubusercontent.com/vLia/" +
			"b3df9e43f55fd43d1bca93cdfd5ae27c/raw/535e8db46f33693a793c616fc1e2b4d77c4b06d2/example-k8s-pod-yaml"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeFileURI),
			Uri:   uri,
		}

		_, err = content.NewFetcher(tempDir).FetchURI(uri)
		assert.NoError(t, err)

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "passed", result.Steps[1].Status)
	})

	t.Run("runner should return failure on yaml gist file URI with deprecated/deleted APIs", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}
		uri := "https://gist.githubusercontent.com/vLia/" +
			"91289de9cc8b6953be5f90b0a52fa8d3/raw/7d49329a16cb2d9d27371553658280a975ca2223/example-k8s-pod-yaml-deprecated"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeFileURI),
			Uri:   uri,
		}

		_, err = content.NewFetcher(tempDir).FetchURI(uri)
		assert.NoError(t, err)

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "failed", result.Steps[1].Status)
	})
}

func TestRunGitFile_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should return error on non-existent Git path", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}

		repo := &testkube.Repository{
			Uri:    "https://github.com/kubeshop/testkube",
			Branch: "develop",
			Path:   "contrib/executor/kubepug/pkg/runner/test-files/fake-manifest.yaml",
		}
		_, err = content.NewFetcher(tempDir).FetchGit(repo)
		assert.NoError(t, err)

		execution.Content = &testkube.TestContent{
			Type_:      string(testkube.TestContentTypeGitFile),
			Repository: repo,
		}

		_, err = runner.Run(ctx, *execution)

		assert.Error(t, err)
	})

	t.Run("runner should return deprecated and deleted APIs on Git file containing deprecated and delete API definitions", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}

		repo := &testkube.Repository{
			Uri:    "https://github.com/kubeshop/testkube",
			Branch: "develop",
			Path:   "contrib/executor/kubepug/pkg/runner/test-files/manifest-deprecated.yaml",
		}
		_, err = content.NewFetcher(tempDir).FetchGit(repo)
		assert.NoError(t, err)

		execution.Content = &testkube.TestContent{
			Type_:      string(testkube.TestContentTypeGitFile),
			Repository: repo,
		}

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "failed", result.Steps[1].Status)
	})
}

func TestRunGitDirectory_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should return success on manifests from Git directory", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}

		repo := &testkube.Repository{
			Uri:    "https://github.com/kubeshop/testkube",
			Branch: "develop",
			Path:   "contrib/executor/kubepug/pkg/runner/test-files/manifest-valid",
		}
		_, err = content.NewFetcher(tempDir).FetchGit(repo)
		assert.NoError(t, err)

		execution.Content = &testkube.TestContent{
			Type_:      string(testkube.TestContentTypeGitDir),
			Repository: repo,
		}

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "passed", result.Steps[1].Status)
	})
}

func TestRunWithSpecificK8sVersion_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should return failure and list of deprecated APIs result "+
		"on yaml containing deprecated API with current K8s version", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: v1
kind: Service
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  ports:
    - name: http
      port: 8001
      targetPort: 80
  selector:
    app: testkube-dashboard
---
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testkube-dashboard
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: testkube-dashboard
    spec:
      containers:
        - name: petstore-backend
          image: docker.io/kubeshop/testkube-dashboard:latest
          ports:
            - name: http
              containerPort: 80
          livenessProbe:
            httpGet:
            path: /index.html
            port: 80
            initialDelaySeconds: 15
            timeoutSeconds: 2
            periodSeconds: 5
            failureThreshold: 1
          readinessProbe:
            httpGet:
              path: /index.html
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 5
            failureThreshold: 1
`

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(data), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write postman runner test content file")
		}

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
		}
		execution.Content = testkube.NewStringTestContent("")

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "failed", result.Steps[1].Status)
	})

	t.Run("runner should return success on yaml containing deprecated API with old K8s version", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: v1
kind: Service
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  ports:
    - name: http
      port: 8001
      targetPort: 80
  selector:
    app: testkube-dashboard
---
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: testkube-dashboard
  namespace: testkube
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testkube-dashboard
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: testkube-dashboard
    spec:
      containers:
        - name: petstore-backend
          image: docker.io/kubeshop/testkube-dashboard:latest
          ports:
            - name: http
              containerPort: 80
          livenessProbe:
            httpGet:
            path: /index.html
            port: 80
            initialDelaySeconds: 15
            timeoutSeconds: 2
            periodSeconds: 5
            failureThreshold: 1
          readinessProbe:
            httpGet:
              path: /index.html
              port: 80
            initialDelaySeconds: 15
            periodSeconds: 5
            failureThreshold: 1
`

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte(data), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write postman runner test content file")
		}

		runner, err := NewRunner(context.Background(), envs.Params{DataDir: tempDir})
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"kubepug"}
		execution.Args = []string{
			"--format=json",
			"--input-file",
			"<runPath>",
			"--k8s-version=v1.7.0", // last version apps/v1beta2/Deployment was valid
		}
		execution.Content = testkube.NewStringTestContent("")

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Equal(t, 2, len(result.Steps))
		assert.Equal(t, "passed", result.Steps[0].Status)
		assert.Equal(t, "passed", result.Steps[1].Status)
	})
}
