package v2

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsuitev2 "github.com/kubeshop/testkube/api/testsuite/v2"
)

const (
	testkubeTestsuiteSecretLabel = "testsuites-secrets"
	currentSecretKey             = "current-secret"
)

var testsuiteSecretDefaultLabels = map[string]string{
	"testkube":           testkubeTestsuiteSecretLabel,
	"testkubeSecretType": "variables",
}

//go:generate mockgen -destination=./mock_testsuites.go -package=v2 "github.com/kubeshop/testkube/pkg/operator/client/testsuites/v2" Interface
type Interface interface {
	List(selector string) (*testsuitev2.TestSuiteList, error)
	ListLabels() (map[string][]string, error)
	Get(name string) (*testsuitev2.TestSuite, error)
	Create(testsuite *testsuitev2.TestSuite) (*testsuitev2.TestSuite, error)
	Update(testsuite *testsuitev2.TestSuite) (*testsuitev2.TestSuite, error)
	Delete(name string) error
	DeleteAll() error
	CreateTestsuiteSecrets(testsuite *testsuitev2.TestSuite) error
	UpdateTestsuiteSecrets(testsuite *testsuitev2.TestSuite) error
	LoadTestsuiteVariablesSecret(testsuite *testsuitev2.TestSuite) (*corev1.Secret, error)
	GetCurrentSecretUUID(testSuiteName string) (string, error)
	GetSecretTestSuiteVars(testSuiteName, secretUUID string) (map[string]string, error)
	DeleteByLabels(selector string) error
	UpdateStatus(testSuite *testsuitev2.TestSuite) error
}

// NewClient creates new TestSuite client
func NewClient(client client.Client, namespace string) *TestSuitesClient {
	return &TestSuitesClient{
		Client:    client,
		Namespace: namespace,
	}
}

// TestSuitesClient implements methods to work with TestSuites
type TestSuitesClient struct {
	Client    client.Client
	Namespace string
}

// List lists TestSuites
func (s TestSuitesClient) List(selector string) (*testsuitev2.TestSuiteList, error) {
	list := &testsuitev2.TestSuiteList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.Namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	if err = s.Client.List(context.Background(), list, options); err != nil {
		return list, err
	}

	for i := range list.Items {
		secret, err := s.LoadTestsuiteVariablesSecret(&list.Items[i])
		secretExists := !errors.IsNotFound(err)
		if err != nil && secretExists {
			return list, err
		}

		secretToTestsuiteVars(secret, &list.Items[i])
	}

	return list, nil
}

// ListLabelslists labels for TestSuites
func (s TestSuitesClient) ListLabels() (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testsuitev2.TestSuiteList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return labels, err
	}

	for _, testsuite := range list.Items {
		for key, value := range testsuite.Labels {
			if values, ok := labels[key]; !ok {
				labels[key] = []string{value}
			} else {
				for _, v := range values {
					if v == value {
						continue
					}
				}
				labels[key] = append(labels[key], value)
			}
		}
	}

	return labels, nil
}

// Get returns TestSuite
func (s TestSuitesClient) Get(name string) (*testsuitev2.TestSuite, error) {
	testsuite := &testsuitev2.TestSuite{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, testsuite)
	if err != nil {
		return nil, err
	}

	secret, err := s.LoadTestsuiteVariablesSecret(testsuite)
	secretExists := !errors.IsNotFound(err)
	if err != nil && secretExists {
		return nil, err
	}

	secretToTestsuiteVars(secret, testsuite)

	return testsuite, nil
}

// Create creates new TestSuite
func (s TestSuitesClient) Create(testsuite *testsuitev2.TestSuite) (*testsuitev2.TestSuite, error) {
	err := s.CreateTestsuiteSecrets(testsuite)
	if err != nil {
		return nil, err
	}

	err = s.Client.Create(context.Background(), testsuite)
	return testsuite, err
}

// Update updates existing TestSuite
func (s TestSuitesClient) Update(testsuite *testsuitev2.TestSuite) (*testsuitev2.TestSuite, error) {
	err := s.UpdateTestsuiteSecrets(testsuite)
	if err != nil {
		return nil, err
	}

	err = s.Client.Update(context.Background(), testsuite)
	return testsuite, err
}

// Delete deletes existing TestSuite
func (s TestSuitesClient) Delete(name string) error {
	testsuite, err := s.Get(name)
	if err != nil {
		return err
	}

	secret, err := s.LoadTestsuiteVariablesSecret(testsuite)
	secretExists := !errors.IsNotFound(err)
	if err != nil && secretExists {
		return err
	}

	if err == nil && secret != nil {
		if err = s.Client.Delete(context.Background(), secret); err != nil {
			return err
		}
	}

	err = s.Client.Delete(context.Background(), testsuite)
	if err != nil {
		return err
	}

	return nil
}

// DeleteAll delete all TestSuites
func (s TestSuitesClient) DeleteAll() error {
	u := &unstructured.Unstructured{}
	u.SetKind("TestSuite")
	u.SetAPIVersion("tests.testkube.io/v2")
	err := s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace))
	if err != nil {
		return err
	}

	u = &unstructured.Unstructured{}
	u.SetKind("Secret")
	u.SetAPIVersion("v1")
	return s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabels(testsuiteSecretDefaultLabels))
}

// CreateTestsuiteSecrets creates corresponding TestSuite vars secrets
func (s TestSuitesClient) CreateTestsuiteSecrets(testsuite *testsuitev2.TestSuite) error {
	secretName := secretName(testsuite.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.Namespace,
			Labels:    testsuiteSecretDefaultLabels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{},
	}

	if err := testVarsToSecret(testsuite, secret); err != nil {
		return err
	}

	if len(secret.StringData) > 0 {
		err := s.Client.Create(context.Background(), secret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s TestSuitesClient) UpdateTestsuiteSecrets(testsuite *testsuitev2.TestSuite) error {
	secret, err := s.LoadTestsuiteVariablesSecret(testsuite)
	secretExists := !errors.IsNotFound(err)
	if err != nil && secretExists {
		return err
	}

	if err == nil && secret == nil {
		return nil
	}

	if !secretExists {
		secret.Name = secretName(testsuite.Name)
		secret.Namespace = s.Namespace
		secret.Labels = testsuiteSecretDefaultLabels
		secret.Type = corev1.SecretTypeOpaque
	}

	if err = testVarsToSecret(testsuite, secret); err != nil {
		return err
	}

	if len(secret.StringData) > 0 {
		if !secretExists {
			err = s.Client.Create(context.Background(), secret)
		} else {
			err = s.Client.Update(context.Background(), secret)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (s TestSuitesClient) TestsuiteHasSecrets(testsuite *testsuitev2.TestSuite) (has bool) {
	if testsuite.Spec.ExecutionRequest == nil {
		return
	}

	for _, v := range testsuite.Spec.ExecutionRequest.Variables {
		if v.Type_ == commonv1.VariableTypeSecret && (v.ValueFrom.SecretKeyRef == nil ||
			(v.ValueFrom.SecretKeyRef != nil && (v.ValueFrom.SecretKeyRef.Name == secretName(testsuite.Name)))) {
			return true
		}
	}

	return
}

func (s TestSuitesClient) LoadTestsuiteVariablesSecret(testsuite *testsuitev2.TestSuite) (*corev1.Secret, error) {
	if !s.TestsuiteHasSecrets(testsuite) {
		return nil, nil
	}
	secret := &corev1.Secret{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: secretName(testsuite.Name)}, secret)
	return secret, err
}

// GetCurrentSecretUUID returns current secret uuid
func (s TestSuitesClient) GetCurrentSecretUUID(testSuiteName string) (string, error) {
	secret := &corev1.Secret{}
	if err := s.Client.Get(context.Background(), client.ObjectKey{
		Namespace: s.Namespace, Name: secretName(testSuiteName)}, secret); err != nil && !errors.IsNotFound(err) {
		return "", err
	}

	secretUUID := ""
	if secret.Data != nil {
		if value, ok := secret.Data[currentSecretKey]; ok {
			secretUUID = string(value)
		}
	}

	return secretUUID, nil
}

// GetSecretTestSuiteVars returns secret test suite vars
func (s TestSuitesClient) GetSecretTestSuiteVars(testSuiteName, secretUUID string) (map[string]string, error) {
	secret := &corev1.Secret{}
	if err := s.Client.Get(context.Background(), client.ObjectKey{
		Namespace: s.Namespace, Name: secretName(testSuiteName)}, secret); err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	secrets := make(map[string]string)
	if secret.Data != nil {
		if value, ok := secret.Data[secretUUID]; ok {
			if err := json.Unmarshal(value, &secrets); err != nil {
				return nil, err
			}
		}
	}

	return secrets, nil
}

// UpdateStatus updates existing TestSuite status
func (s TestSuitesClient) UpdateStatus(testSuite *testsuitev2.TestSuite) error {
	return s.Client.Status().Update(context.Background(), testSuite)
}

// testVarsToSecret loads secrets data passed into TestSuite CRD and remove plain text data
func testVarsToSecret(testsuite *testsuitev2.TestSuite, secret *corev1.Secret) error {
	if secret == nil {
		return nil
	}

	if secret.StringData == nil {
		secret.StringData = map[string]string{}
	}

	if testsuite.Spec.ExecutionRequest == nil {
		return nil
	}

	secretMap := make(map[string]string)
	for k := range testsuite.Spec.ExecutionRequest.Variables {
		v := testsuite.Spec.ExecutionRequest.Variables[k]
		if v.Type_ == commonv1.VariableTypeSecret {
			if v.ValueFrom.SecretKeyRef != nil {
				v.ValueFrom = corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: v.ValueFrom.SecretKeyRef.Key,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: v.ValueFrom.SecretKeyRef.Name,
						},
					},
				}
			} else {
				secret.StringData[v.Name] = v.Value
				secretMap[v.Name] = v.Value
				// clear passed test variable secret value and save as reference o secret
				v.Value = ""
				v.ValueFrom = corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: v.Name,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.Name,
						},
					},
				}
			}

			testsuite.Spec.ExecutionRequest.Variables[k] = v
		}
	}

	if len(secretMap) != 0 {
		random, err := uuid.NewRandom()
		if err != nil {
			return err
		}

		data, err := json.Marshal(secretMap)
		if err != nil {
			return err
		}

		secret.StringData[random.String()] = string(data)
		secret.StringData[currentSecretKey] = random.String()
	}

	return nil
}

// secretToTestsuiteVars loads secrets data passed into TestSuite CRD and remove plain text data
func secretToTestsuiteVars(secret *corev1.Secret, testsuite *testsuitev2.TestSuite) {
	if testsuite == nil || secret == nil || secret.Data == nil {
		return
	}

	if testsuite.Spec.ExecutionRequest == nil {
		return
	}

	for k, v := range secret.Data {
		if variable, ok := testsuite.Spec.ExecutionRequest.Variables[k]; ok {
			variable.Value = string(v)
			testsuite.Spec.ExecutionRequest.Variables[k] = variable
		}
	}
}

func secretName(testsuiteName string) string {
	return fmt.Sprintf("%s-testsuitevars", testsuiteName)
}

// DeleteByLabels deletes TestSuites by labels
func (s TestSuitesClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestSuite")
	u.SetAPIVersion("tests.testkube.io/v2")
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
