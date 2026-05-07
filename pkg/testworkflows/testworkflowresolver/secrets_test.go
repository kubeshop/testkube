package testworkflowresolver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func testSecret(name, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: key},
	}
}

func testGitCreate(username, token, sshKey string) *testworkflowsv1.Content {
	var usernameFrom, tokenFrom, sshKeyFrom *corev1.EnvVarSource
	if username != "" {
		usernameFrom = testSecret(ComputedKeyword, username)
	}
	if token != "" {
		tokenFrom = testSecret(ComputedKeyword, token)
	}
	if sshKey != "" {
		sshKeyFrom = testSecret(ComputedKeyword, sshKey)
	}
	return &testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			UsernameFrom: usernameFrom,
			TokenFrom:    tokenFrom,
			SshKeyFrom:   sshKeyFrom,
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

func TestExtract_ContentUserToken(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("some-username", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(testSecret("some-secret-1", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ContentTokenOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "some-token", ""),
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
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ContentSshOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
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
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_StepContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Steps: []testworkflowsv1.Step{
				{StepSource: testworkflowsv1.StepSource{Content: testGitCreate("some-username", "some-token", "")}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.Step{
				{StepSource: testworkflowsv1.StepSource{Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil)}},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ParallelContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Steps: []testworkflowsv1.Step{
				{Parallel: &testworkflowsv1.StepParallel{
					Content: testGitCreate("some-username", "some-token", ""),
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
					Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil),
				}},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ServicesContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Services: map[string]testworkflowsv1.ServiceSpec{
				"some": {
					IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
						Content: testGitCreate("some-username", "some-token", ""),
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
						Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil),
					},
				},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

// Test Workflow Templates

func TestExtractTemplate_ContentUserToken(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("some-username", "some-token", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(testSecret("some-secret-1", GitUsernameKey), testSecret("some-secret-2", GitTokenKey), nil),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_ContentTokenOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "some-token", ""),
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
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_ContentSshOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
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
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_StepContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepSource: testworkflowsv1.StepSource{Content: testGitCreate("some-username", "some-token", "")}},
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGit(nil, nil, testSecret("some-secret-1", GitSshKey)),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{StepSource: testworkflowsv1.StepSource{Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil)}},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_ParallelContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Steps: []testworkflowsv1.IndependentStep{
				{Parallel: &testworkflowsv1.IndependentStepParallel{
					TestWorkflowTemplateSpec: testworkflowsv1.TestWorkflowTemplateSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content: testGitCreate("some-username", "some-token", ""),
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
							Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil),
						},
					},
				}},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}

func testGitCreateCerts(caCert, clientCert, clientKey string) *testworkflowsv1.Content {
	var caCertFrom, clientCertFrom, clientKeyFrom *corev1.EnvVarSource
	if caCert != "" {
		caCertFrom = testSecret(ComputedKeyword, caCert)
	}
	if clientCert != "" {
		clientCertFrom = testSecret(ComputedKeyword, clientCert)
	}
	if clientKey != "" {
		clientKeyFrom = testSecret(ComputedKeyword, clientKey)
	}
	return &testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			CaCertFrom:     caCertFrom,
			ClientCertFrom: clientCertFrom,
			ClientKeyFrom:  clientKeyFrom,
		},
	}
}

func testGitCerts(caCert, clientCert, clientKey *corev1.EnvVarSource) *testworkflowsv1.Content {
	return &testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			CaCertFrom:     caCert,
			ClientCertFrom: clientCert,
			ClientKeyFrom:  clientKey,
		},
	}
}

func TestExtract_ContentCaCertOnly(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreateCerts("some-ca-cert", "", ""),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCerts(testSecret("some-secret-1", GitCaCertKey), nil, nil),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitCaCertKey, "some-ca-cert"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ContentClientCertAndKey(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreateCerts("", "some-client-cert", "some-client-key"),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCerts(nil, testSecret("some-secret-1", GitClientCertKey), testSecret("some-secret-2", GitClientKey)),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitClientCertKey, "some-client-cert"}, {GitClientKey, "some-client-key"}}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtract_ContentAllCerts(t *testing.T) {
	wf := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreateCerts("some-ca-cert", "some-client-cert", "some-client-key"),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCerts(
					testSecret("some-secret-1", GitCaCertKey),
					testSecret("some-secret-2", GitClientCertKey),
					testSecret("some-secret-3", GitClientKey),
				),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInWorkflow(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{
		{GitCaCertKey, "some-ca-cert"},
		{GitClientCertKey, "some-client-cert"},
		{GitClientKey, "some-client-key"},
	}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_ContentAllCerts(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreateCerts("some-ca-cert", "some-client-cert", "some-client-key"),
			},
		},
	}
	expected := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCerts(
					testSecret("some-secret-1", GitCaCertKey),
					testSecret("some-secret-2", GitClientCertKey),
					testSecret("some-secret-3", GitClientKey),
				),
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{
		{GitCaCertKey, "some-ca-cert"},
		{GitClientCertKey, "some-client-cert"},
		{GitClientKey, "some-client-key"},
	}, calls)
	assert.Equal(t, expected, wf)
}

func TestExtractTemplate_ServicesContent(t *testing.T) {
	wf := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: testGitCreate("", "", "some-key"),
			},
			Services: map[string]testworkflowsv1.IndependentServiceSpec{
				"some": {
					Content: testGitCreate("some-username", "some-token", ""),
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
					Content: testGit(testSecret("some-secret-2", GitUsernameKey), testSecret("some-secret-3", GitTokenKey), nil),
				},
			},
		},
	}
	i := 0
	calls := make([][]string, 0)
	err := ExtractCredentialsInTemplate(&wf, func(key, value string) (*corev1.EnvVarSource, error) {
		i++
		calls = append(calls, []string{key, value})
		return testSecret(fmt.Sprintf("some-secret-%d", i), key), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, [][]string{{GitSshKey, "some-key"}, {GitUsernameKey, "some-username"}, {GitTokenKey, "some-token"}}, calls)
	assert.Equal(t, expected, wf)
}
