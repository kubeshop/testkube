package presets

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestNewOpenSourceWithLocalArtifactsInjectsRelayOnlyIntoArtifactStage(t *testing.T) {
	const (
		relayURL    = "http://local-artifact-relay:8080/upload"
		secretName  = "local-artifact-token"
		secretValue = "must-not-appear-in-job-annotations"
	)
	pure := true
	processor := NewOpenSourceWithLocalArtifacts(ins, relayURL, secretName)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{PureByDefault: &pure},
			},
			Steps: []testworkflowsv1.Step{{
				StepMeta: testworkflowsv1.StepMeta{Pure: &pure},
				StepOperations: testworkflowsv1.StepOperations{
					Shell: "echo workflow-step",
					Artifacts: &testworkflowsv1.StepArtifacts{
						Paths: []string{"test-results/**/*"},
					},
				},
			}},
		},
	}

	bundle, err := processor.Bundle(context.Background(), workflow, testworkflowprocessor.BundleOptions{
		Config: testConfig,
		Secrets: []corev1.Secret{{
			ObjectMeta: metav1.ObjectMeta{Name: secretName},
			StringData: map[string]string{localartifacts.TokenSecretKey: secretValue},
		}},
	})
	require.NoError(t, err)
	// The local-artifact processor must deep-copy before forcing isolation, so
	// callers retain their original TestWorkflow object and can reuse it.
	require.Nil(t, workflow.Spec.System.IsolatedContainers)

	assertLocalArtifactTokenIsolated(t, bundle, secretName, relayURL, "workflow-step")
	require.True(t, podUsesRelayToken(bundle.Job.Spec.Template.Spec, secretName), "the workflow Pod must receive the relay Secret only as an environment reference")

	annotations := bundle.Job.Spec.Template.Annotations
	require.NotContains(t, annotations[constants.InternalAnnotationName], secretValue)
	require.NotContains(t, annotations[constants.SpecAnnotationName], secretValue)
}

func TestNewOpenSourceWithLocalArtifactsIsolatesPureByDefaultUserShell(t *testing.T) {
	const (
		relayURL   = "http://local-artifact-relay:8080/upload"
		secretName = "local-artifact-token"
	)
	pureByDefault := true
	processor := NewOpenSourceWithLocalArtifacts(ins, relayURL, secretName)
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				System: &testworkflowsv1.TestWorkflowSystem{PureByDefault: &pureByDefault},
			},
			Steps: []testworkflowsv1.Step{{
				StepOperations: testworkflowsv1.StepOperations{
					Shell:     "echo pure-by-default-user-shell",
					Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"test-results/**/*"}},
				},
			}},
		},
	}

	bundle, err := processor.Bundle(context.Background(), workflow, testworkflowprocessor.BundleOptions{Config: testConfig})
	require.NoError(t, err)
	require.Nil(t, workflow.Spec.System.IsolatedContainers)
	assertLocalArtifactTokenIsolated(t, bundle, secretName, relayURL, "pure-by-default-user-shell")
}

func assertLocalArtifactTokenIsolated(t *testing.T, bundle *testworkflowprocessor.Bundle, secretName, relayURL, userShellMarker string) {
	t.Helper()
	artifactStageCount := 0
	for _, group := range bundle.Actions() {
		containsArtifact := false
		containsUserShell := false
		for _, action := range group {
			if action.Container == nil {
				continue
			}
			if action.Container.Config.Command != nil && contains(*action.Container.Config.Command, "artifacts") {
				containsArtifact = true
			}
			if action.Container.Config.Args != nil && strings.Contains(strings.Join(*action.Container.Config.Args, " "), userShellMarker) {
				containsUserShell = true
			}
		}
		if containsArtifact {
			require.False(t, containsUserShell, "the relay Secret must not share an action group with user shell code")
		}
		for _, action := range group {
			if action.Container == nil || action.Container.Config.Command == nil {
				continue
			}
			command := *action.Container.Config.Command
			isArtifactStage := contains(command, "artifacts")
			for _, env := range action.Container.Config.Env {
				if env.Name != localartifacts.TokenEnvName {
					continue
				}
				require.True(t, isArtifactStage, "only the artifacts stage may receive the relay token")
				require.NotNil(t, env.ValueFrom)
				require.NotNil(t, env.ValueFrom.SecretKeyRef)
				require.Equal(t, secretName, env.ValueFrom.SecretKeyRef.Name)
				require.Equal(t, localartifacts.TokenSecretKey, env.ValueFrom.SecretKeyRef.Key)
			}
			if !isArtifactStage {
				continue
			}
			artifactStageCount++
			require.Contains(t, command, "--local-upload-url")
			require.Contains(t, command, relayURL)
		}
	}
	require.Equal(t, 1, artifactStageCount)
}

func TestNewOpenSourceWithLocalArtifactsRejectsInvalidRelayConfiguration(t *testing.T) {
	processor := NewOpenSourceWithLocalArtifacts(ins, "ftp://relay", "")
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{{
				StepOperations: testworkflowsv1.StepOperations{
					Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"result.txt"}},
				},
			}},
		},
	}

	_, err := processor.Bundle(context.Background(), workflow, testworkflowprocessor.BundleOptions{Config: testConfig})
	require.ErrorContains(t, err, "local artifact upload URL")
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestLocalArtifactsDoesNotExposeTokenThroughStringData(t *testing.T) {
	// This small regression guard is intentionally separate from the generated
	// bundle assertion above: the stage carries a SecretKeyRef, never a literal
	// token value.
	processor := NewOpenSourceWithLocalArtifacts(ins, "http://relay:8080/upload", "relay-token")
	workflow := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{{
			StepOperations: testworkflowsv1.StepOperations{Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"result.txt"}}},
		}}},
	}
	bundle, err := processor.Bundle(context.Background(), workflow, testworkflowprocessor.BundleOptions{Config: testConfig})
	require.NoError(t, err)
	require.True(t, podUsesRelayToken(bundle.Job.Spec.Template.Spec, "relay-token"))
	require.NotContains(t, bundle.Job.Spec.Template.Annotations[constants.InternalAnnotationName], localartifacts.TokenEnvName)
}

func podUsesRelayToken(spec corev1.PodSpec, secretName string) bool {
	containers := append([]corev1.Container{}, spec.InitContainers...)
	containers = append(containers, spec.Containers...)
	for _, container := range containers {
		for _, env := range container.Env {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				continue
			}
			if env.ValueFrom.SecretKeyRef.Name == secretName && env.ValueFrom.SecretKeyRef.Key == localartifacts.TokenSecretKey {
				return env.Value == ""
			}
		}
	}
	return false
}
