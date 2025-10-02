package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv2 "github.com/kubeshop/testkube/api/tests/v2"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testkubeTestSecretLabel = "tests-secrets"
	currentSecretKey        = "current-secret"
)

var testSecretDefaultLabels = map[string]string{
	"testkube":           testkubeTestSecretLabel,
	"testkubeSecretType": "variables",
}

// NewClent creates new Test client
func NewClient(client client.Client, namespace string) *TestsClient {
	return &TestsClient{
		Client:    client,
		Namespace: namespace,
	}
}

// TestClient implements methods to work with Test
type TestsClient struct {
	Client    client.Client
	Namespace string
}

// List lists Tests
func (s TestsClient) List(selector string) (*testsv2.TestList, error) {

	list := &testsv2.TestList{}
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
		secret, err := s.LoadTestVariablesSecret(&list.Items[i])
		if err != nil && !s.ErrIsNotFound(err) {
			return list, err
		}

		secretToTestVars(secret, &list.Items[i])
	}

	return list, nil
}

// ListLabels labels for Tests
func (s TestsClient) ListLabels() (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testsv2.TestList{}
	if err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace}); err != nil {
		return labels, err
	}

	for _, test := range list.Items {
		for key, value := range test.Labels {
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

// Get returns Test, loads and decodes secrets data
func (s TestsClient) Get(name string) (*testsv2.Test, error) {
	test := &testsv2.Test{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, test)
	if err != nil {
		return nil, err
	}

	secret, err := s.LoadTestVariablesSecret(test)
	if err != nil && !s.ErrIsNotFound(err) {
		return nil, err
	}

	secretToTestVars(secret, test)

	return test, nil
}

func (s TestsClient) ErrIsNotFound(err error) bool {
	if err != nil {
		return strings.Contains(err.Error(), "not found")
	}
	return false
}

// Create creates new Test and coupled resources
func (s TestsClient) Create(test *testsv2.Test) (*testsv2.Test, error) {
	err := s.CreateTestSecrets(test)
	if err != nil {
		return nil, err
	}
	err = s.Client.Create(context.Background(), test)
	return test, err
}

// Update updates existing Test and coupled resources
func (s TestsClient) Update(test *testsv2.Test) (*testsv2.Test, error) {
	err := s.UpdateTestSecrets(test)
	if err != nil {
		return nil, err
	}
	err = s.Client.Update(context.Background(), test)
	return test, err
}

// Delete deletes existing Test and coupled resources (secrets)
func (s TestsClient) Delete(name string) error {
	test, err := s.Get(name)
	if err != nil {
		return err
	}

	secret, err := s.LoadTestVariablesSecret(test)
	secretExists := !s.ErrIsNotFound(err)
	if err != nil && secretExists {
		return err
	}

	err = s.Client.Delete(context.Background(), test)
	if err != nil {
		return err
	}

	// delete secret only if exists ignore otherwise
	if secretExists && secret != nil {
		err = s.Client.Delete(context.Background(), secret)
		return err
	}

	return nil
}

// DeleteAll deletes all Tests
func (s TestsClient) DeleteAll() error {

	u := &unstructured.Unstructured{}
	u.SetKind("Test")
	u.SetAPIVersion("tests.testkube.io/v2")
	err := s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace))
	if err != nil {
		return err
	}

	u = &unstructured.Unstructured{}
	u.SetKind("Secret")
	u.SetAPIVersion("v1")
	return s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabels(testSecretDefaultLabels))
}

// CreateTestSecrets creates corresponding test vars secrets
func (s TestsClient) CreateTestSecrets(test *testsv2.Test) error {
	secretName := secretName(test.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.Namespace,
			Labels:    testSecretDefaultLabels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{},
	}

	if err := testVarsToSecret(test, secret); err != nil {
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

func (s TestsClient) UpdateTestSecrets(test *testsv2.Test) error {
	secret, err := s.LoadTestVariablesSecret(test)
	if err != nil && !s.ErrIsNotFound(err) {
		return err
	}

	if secret == nil {
		return nil
	}

	if err := testVarsToSecret(test, secret); err != nil {
		return err
	}

	if len(secret.StringData) > 0 {
		err := s.Client.Update(context.Background(), secret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s TestsClient) TestHasSecrets(test *testsv2.Test) (has bool) {
	for _, v := range test.Spec.Variables {
		if v.Type_ == commonv1.VariableTypeSecret {
			return true
		}
	}

	return
}

func (s TestsClient) LoadTestVariablesSecret(test *testsv2.Test) (*corev1.Secret, error) {
	if !s.TestHasSecrets(test) {
		return nil, nil
	}
	secret := &corev1.Secret{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: secretName(test.Name)}, secret)
	return secret, err
}

// GetCurrentSecretUUID returns current secret uuid
func (s TestsClient) GetCurrentSecretUUID(testName string) (string, error) {
	secret := &corev1.Secret{}
	if err := s.Client.Get(context.Background(), client.ObjectKey{
		Namespace: s.Namespace, Name: secretName(testName)}, secret); err != nil && !s.ErrIsNotFound(err) {
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

// GetSecretTestVars returns secret test vars
func (s TestsClient) GetSecretTestVars(testName, secretUUID string) (map[string]string, error) {
	secret := &corev1.Secret{}
	if err := s.Client.Get(context.Background(), client.ObjectKey{
		Namespace: s.Namespace, Name: secretName(testName)}, secret); err != nil && !s.ErrIsNotFound(err) {
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

// ListByNames returns Tests by names
// TODO - should be replaced by --field-selector when it supports IN for expression
func (s TestsClient) ListByNames(names []string) ([]testsv2.Test, error) {
	tests := []testsv2.Test{}
	for _, name := range names {
		test := &testsv2.Test{}
		if err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, test); err != nil {
			return nil, err
		}

		tests = append(tests, *test)
	}

	return tests, nil
}

// testVarsToSecret loads secrets data passed into Test CRD and remove plain text data
func testVarsToSecret(test *testsv2.Test, secret *corev1.Secret) error {
	if secret.StringData == nil {
		secret.StringData = map[string]string{}
	}

	secretMap := make(map[string]string)
	for k := range test.Spec.Variables {
		v := test.Spec.Variables[k]
		if v.Type_ == commonv1.VariableTypeSecret {
			secret.StringData[v.Name] = v.Value
			secretMap[v.Name] = v.Value
			// clear passed test variable secret value and save as reference o secret
			v.Value = ""
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
				v.ValueFrom = corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: v.Name,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.Name,
						},
					},
				}
			}

			test.Spec.Variables[k] = v
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

// secretToTestVars loads secrets data passed into Test CRD and remove plain text data
func secretToTestVars(secret *corev1.Secret, test *testsv2.Test) {
	if test == nil || secret == nil || secret.Data == nil {
		return
	}

	for k, v := range secret.Data {
		if variable, ok := test.Spec.Variables[k]; ok {
			variable.Value = string(v)
			test.Spec.Variables[k] = variable
		}
	}
}

func secretName(testName string) string {
	return fmt.Sprintf("%s-testvars", testName)
}

// DeleteByLabels deletes tests by labels
func (s TestsClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("Test")
	u.SetAPIVersion("tests.testkube.io/v2")
	err = s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}
