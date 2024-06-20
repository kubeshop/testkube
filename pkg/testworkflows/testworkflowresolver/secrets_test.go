package testworkflowresolver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

func testSecret(name, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: key},
	}
}

func testGitPlain(username, token, sshKey string) *testworkflowsv1.Content {
	return &testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			Username: username,
			Token:    token,
			SshKey:   sshKey,
		},
	}
}

func testGit(username, token, sshKey *corev1.EnvVarSource) *testworkflowsv1.Content {
	return &testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			UsernameFrom: username,
			TokenFrom:    token,
			SshKeyFrom:   sshKey,
		},
	}
}

// Test Workflows

func TestReplacePlainText_ContentUserToken(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("some-username", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(testSecret("some-secret-1", GitUsernameKey), testSecret("some-secret-1", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainText_ContentTokenOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, testSecret("some-secret-1", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainText_ContentSshOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainText_StepContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Steps: []testworkflowsv1.Step{
				{StepSource: testworkflowsv1.StepSource{Content: testGitPlain("some-username", "some-token", "")}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.Step{
				{StepSource: testworkflowsv1.StepSource{Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil)}},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainText_ParallelContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Steps: []testworkflowsv1.Step{
				{Parallel: &testworkflowsv1.StepParallel{
					TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content: testGitPlain("some-username", "some-token", ""),
						},
					},
				}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.Step{
				{Parallel: &testworkflowsv1.StepParallel{
					TestWorkflowSpec: testworkflowsv1.TestWorkflowSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
						},
					},
				}},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainText_ServicesContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Services: map[string]testworkflowsv1.ServiceSpec{
				"some": {
					IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
						Content: testGitPlain("some-username", "some-token", ""),
					},
				},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Services: map[string]testworkflowsv1.ServiceSpec{
				"some": {
					IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
						Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
					},
				},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInWorkflow(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

// Test Workflow Templates

func TestReplacePlainTextTemplate_ContentUserToken(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("some-username", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(testSecret("some-secret-1", GitUsernameKey), testSecret("some-secret-1", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainTextTemplate_ContentTokenOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, testSecret("some-secret-1", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainTextTemplate_ContentSshOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainTextTemplate_StepContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepSource: testworkflowsv1.StepSource{Content: testGitPlain("some-username", "some-token", "")}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepSource: testworkflowsv1.StepSource{Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil)}},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainTextTemplate_ParallelContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{Parallel: &testworkflowsv1.IndependentStepParallel{
					TestWorkflowTemplateSpec: testworkflowsv1.TestWorkflowTemplateSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content: testGitPlain("some-username", "some-token", ""),
						},
					},
				}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{Parallel: &testworkflowsv1.IndependentStepParallel{
					TestWorkflowTemplateSpec: testworkflowsv1.TestWorkflowTemplateSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
						},
					},
				}},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestReplacePlainTextTemplate_ServicesContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitPlain("", "", "some-key"),
			},
			Services: map[string]testworkflowsv1.IndependentServiceSpec{
				"some": {
					Content: testGitPlain("some-username", "some-token", ""),
				},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Services: map[string]testworkflowsv1.IndependentServiceSpec{
				"some": {
					Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
				},
			},
		},
	}
	i := 0
	calls := make([]map[string]string, 0)
	secrets, err := ReplacePlainTextCredentialsInTemplate(&wf, func(creds map[string]string) (string, error) {
		i++
		calls = append(calls, creds)
		return fmt.Sprintf("some-secret-%d", i), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"some-secret-1", "some-secret-2"}, secrets)
	assert.Equal(t, []map[string]string{{GitSshKey: "some-key"}, {GitUsernameKey: "some-username", GitTokenKey: "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}
