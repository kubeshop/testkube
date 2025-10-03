package tests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/labels"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	"github.com/kubeshop/testkube/pkg/operator/secret"
)

const (
	testkubeTestSecretLabel = "tests-secrets"
	currentSecretKey        = "current-secret"
	// secretKind is a kind of the secrets
	secretKind = "secrets"
	// gitUsernameSecretName is git username secret name
	gitUsernameSecretName = "git-username"
	// gitTokenSecretName is git token secret name
	gitTokenSecretName = "git-token"
)

var testSecretDefaultLabels = map[string]string{
	"testkube":           testkubeTestSecretLabel,
	"testkubeSecretType": "variables",
	"createdBy":          "testkube",
}

//go:generate mockgen -source=./test.go -destination=./mock_tests.go -package=tests "github.com/kubeshop/testkube/pkg/operator/client/tests/v3" Interface
type Interface interface {
	List(selector string) (*testsv3.TestList, error)
	ListLabels() (map[string][]string, error)
	Get(name string) (*testsv3.Test, error)
	Create(test *testsv3.Test, disableSecretCreation bool, options ...Option) (*testsv3.Test, error)
	Update(test *testsv3.Test, disableSecretCreation bool, options ...Option) (*testsv3.Test, error)
	Delete(name string) error
	DeleteAll() error
	CreateTestSecrets(test *testsv3.Test, disableSecretCreation bool) error
	UpdateTestSecrets(test *testsv3.Test, disableSecretCreation bool) error
	LoadTestVariablesSecret(test *testsv3.Test) (*corev1.Secret, error)
	GetCurrentSecretUUID(testName string) (string, error)
	GetSecretTestVars(testName, secretUUID string) (map[string]string, error)
	ListByNames(names []string) ([]testsv3.Test, error)
	DeleteByLabels(selector string) error
	UpdateStatus(test *testsv3.Test) error
}

type DeleteDependenciesError struct {
	testName  string
	allErrors []error
}

func (e *DeleteDependenciesError) Error() string {
	return fmt.Errorf("removing dependencies of test %s returned errors: %v", e.testName, e.allErrors).Error()
}

func NewDeleteDependenciesError(testName string, allErrors []error) error {
	return &DeleteDependenciesError{testName: testName, allErrors: allErrors}
}

// Option contain test options
type Option struct {
	Secrets map[string]string
}

// NewClient creates new Test client
func NewClient(client client.Client, namespace string) *TestsClient {
	return &TestsClient{
		k8sClient:    client,
		namespace:    namespace,
		secretClient: secret.NewClient(client, namespace, secret.TestkubeTestSecretLabel),
	}
}

// TestsClient implements methods to work with Test
type TestsClient struct {
	k8sClient    client.Client
	namespace    string
	secretClient *secret.Client
}

// List lists Tests
func (s TestsClient) List(selector string) (*testsv3.TestList, error) {
	list := &testsv3.TestList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}
	if err = s.k8sClient.List(context.Background(), list, options); err != nil {
		return list, err
	}

	for i := range list.Items {
		secret, err := s.LoadTestVariablesSecret(&list.Items[i])
		if err != nil && !errors.IsNotFound(err) {
			return list, err
		}

		secretToTestVars(secret, &list.Items[i])
	}

	return list, nil
}

// ListLabels labels for Tests
func (s TestsClient) ListLabels() (map[string][]string, error) {
	labels := map[string][]string{}
	list := &testsv3.TestList{}
	if err := s.k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: s.namespace}); err != nil {
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
func (s TestsClient) Get(name string) (*testsv3.Test, error) {
	test := &testsv3.Test{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, test)
	if err != nil {
		return nil, err
	}

	secret, err := s.LoadTestVariablesSecret(test)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	secretToTestVars(secret, test)

	return test, nil
}

// Create creates new Test and coupled resources
func (s TestsClient) Create(test *testsv3.Test, disableSecretCreation bool, options ...Option) (*testsv3.Test, error) {
	err := s.CreateTestSecrets(test, disableSecretCreation)
	if err != nil {
		return nil, err
	}

	if len(options) != 0 {
		secrets := make(map[string]string, 0)
		for _, option := range options {
			for key, value := range option.Secrets {
				secrets[key] = value
			}
		}

		if len(secrets) != 0 {
			secretName := secret.GetMetadataName(test.Name, secretKind)
			if err := s.secretClient.Create(secretName, test.Labels, secrets); err != nil {
				return nil, err
			}

			updateTestSecrets(test, secretName, secrets)
		}
	}

	err = s.k8sClient.Create(context.Background(), test)
	return test, err
}

// Update updates existing Test and coupled resources
func (s TestsClient) Update(test *testsv3.Test, disableSecretCreation bool, options ...Option) (*testsv3.Test, error) {
	err := s.UpdateTestSecrets(test, disableSecretCreation)
	if err != nil {
		return nil, err
	}

	if len(options) != 0 {
		secrets := make(map[string]string, 0)
		for _, option := range options {
			for key, value := range option.Secrets {
				secrets[key] = value
			}
		}

		secretName := secret.GetMetadataName(test.Name, secretKind)
		if len(secrets) != 0 {
			if err := s.secretClient.Apply(secretName, test.Labels, secrets); err != nil {
				return nil, err
			}

			updateTestSecrets(test, secretName, secrets)
		} else {
			if err := s.secretClient.Delete(secretName); err != nil && !errors.IsNotFound(err) {
				return nil, err
			}

			clearTestSecrets(test, secretName)
		}
	}

	err = s.k8sClient.Update(context.Background(), test)
	return test, err
}

// Delete deletes existing Test and coupled resources (secrets)
func (s TestsClient) Delete(name string) error {
	test, err := s.Get(name)
	if err != nil {
		return err
	}

	err = s.k8sClient.Delete(context.Background(), test)
	if err != nil {
		return err
	}

	var allErrors []error

	secretObj, err := s.LoadTestVariablesSecret(test)
	if err != nil && !errors.IsNotFound(err) {
		allErrors = append(allErrors, err)
	}

	// delete secret only if exists ignore otherwise
	if err == nil && secretObj != nil {
		err = s.k8sClient.Delete(context.Background(), secretObj)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	secretName := secret.GetMetadataName(test.Name, secretKind)
	if err := s.secretClient.Delete(secretName); err != nil && !errors.IsNotFound(err) {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) != 0 {
		return NewDeleteDependenciesError(name, allErrors)
	}

	return nil
}

// DeleteAll deletes all Tests
func (s TestsClient) DeleteAll() error {
	u := &unstructured.Unstructured{}
	u.SetKind("Secret")
	u.SetAPIVersion("v1")
	err := s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace),
		client.MatchingLabels(testSecretDefaultLabels))
	if err != nil {
		return err
	}

	if err := s.secretClient.DeleteAll(""); err != nil {
		return err
	}

	u = &unstructured.Unstructured{}
	u.SetKind("Test")
	u.SetAPIVersion("tests.testkube.io/v3")
	return s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace))
}

// CreateTestSecrets creates corresponding test vars secrets
func (s TestsClient) CreateTestSecrets(test *testsv3.Test, disableSecretCreation bool) error {
	secretName := secretName(test.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.namespace,
			Labels:    testSecretDefaultLabels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{},
	}

	for key, value := range test.Labels {
		secret.Labels[key] = value
	}

	if err := testVarsToSecret(test, secret, disableSecretCreation); err != nil {
		return err
	}

	if len(secret.StringData) > 0 {
		err := s.k8sClient.Create(context.Background(), secret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s TestsClient) UpdateTestSecrets(test *testsv3.Test, disableSecretCreation bool) error {
	secret, err := s.LoadTestVariablesSecret(test)
	secretExists := !errors.IsNotFound(err)
	if err != nil && secretExists {
		return err
	}

	if err == nil && secret == nil {
		return nil
	}

	if !secretExists {
		secret.Name = secretName(test.Name)
		secret.Namespace = s.namespace
		secret.Labels = testSecretDefaultLabels
		secret.Type = corev1.SecretTypeOpaque
	}

	for key, value := range test.Labels {
		secret.Labels[key] = value
	}

	if err = testVarsToSecret(test, secret, disableSecretCreation); err != nil {
		return err
	}

	if len(secret.StringData) > 0 {
		if !secretExists {
			err = s.k8sClient.Create(context.Background(), secret)
		} else {
			err = s.k8sClient.Update(context.Background(), secret)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (s TestsClient) TestHasSecrets(test *testsv3.Test) (has bool) {
	if test.Spec.ExecutionRequest == nil {
		return
	}

	for _, v := range test.Spec.ExecutionRequest.Variables {
		if v.Type_ == commonv1.VariableTypeSecret && (v.ValueFrom.SecretKeyRef == nil ||
			(v.ValueFrom.SecretKeyRef != nil && (v.ValueFrom.SecretKeyRef.Name == secretName(test.Name)))) {
			return true
		}
	}

	return
}

func (s TestsClient) LoadTestVariablesSecret(test *testsv3.Test) (*corev1.Secret, error) {
	if !s.TestHasSecrets(test) {
		return nil, nil
	}
	secret := &corev1.Secret{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: secretName(test.Name)}, secret)
	return secret, err
}

// GetCurrentSecretUUID returns current secret uuid
func (s TestsClient) GetCurrentSecretUUID(testName string) (string, error) {
	secret := &corev1.Secret{}
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: s.namespace, Name: secretName(testName)}, secret); err != nil && !errors.IsNotFound(err) {
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
	if err := s.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: s.namespace, Name: secretName(testName)}, secret); err != nil && !errors.IsNotFound(err) {
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
func (s TestsClient) ListByNames(names []string) ([]testsv3.Test, error) {
	tests := []testsv3.Test{}
	for _, name := range names {
		test := &testsv3.Test{}
		if err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, test); err != nil {
			return nil, err
		}

		tests = append(tests, *test)
	}

	return tests, nil
}

// UpdateStatus updates existing Test status
func (s TestsClient) UpdateStatus(test *testsv3.Test) error {
	return s.k8sClient.Status().Update(context.Background(), test)
}

// testVarsToSecret loads secrets data passed into Test CRD and remove plain text data
func testVarsToSecret(test *testsv3.Test, secret *corev1.Secret, disablesecretCreation bool) error {
	if secret.StringData == nil {
		secret.StringData = map[string]string{}
	}

	if test.Spec.ExecutionRequest == nil {
		return nil
	}

	secretMap := make(map[string]string)
	for k := range test.Spec.ExecutionRequest.Variables {
		v := test.Spec.ExecutionRequest.Variables[k]
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
				if !disablesecretCreation {
					// save as reference to secret
					secret.StringData[v.Name] = v.Value
					secretMap[v.Name] = v.Value
					v.ValueFrom = corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							Key: v.Name,
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
						},
					}
				}

				// clear passed test variable secret value
				v.Value = ""
			}

			test.Spec.ExecutionRequest.Variables[k] = v
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
func secretToTestVars(secret *corev1.Secret, test *testsv3.Test) {
	if test == nil || secret == nil || secret.Data == nil {
		return
	}

	if test.Spec.ExecutionRequest == nil {
		return
	}

	for k, v := range secret.Data {
		if variable, ok := test.Spec.ExecutionRequest.Variables[k]; ok {
			variable.Value = string(v)
			test.Spec.ExecutionRequest.Variables[k] = variable
		}
	}
}

func secretName(testName string) string {
	return fmt.Sprintf("%s-testvars", testName)
}

// DeleteByLabels deletes tests by labels
func (s TestsClient) DeleteByLabels(selector string) error {
	filter := ""
	for key, value := range testSecretDefaultLabels {
		if filter != "" {
			filter += ","
		}

		filter += fmt.Sprintf("%s=%s", key, value)
	}

	if selector != "" {
		if filter != "" {
			filter += ","
		}

		filter += selector
	}

	reqs, err := labels.ParseToRequirements(filter)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("Secret")
	u.SetAPIVersion("v1")
	err = s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	if err != nil {
		return err
	}

	if err := s.secretClient.DeleteAll(selector); err != nil {
		return err
	}

	reqs, err = labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	u = &unstructured.Unstructured{}
	u.SetKind("Test")
	u.SetAPIVersion("tests.testkube.io/v3")
	err = s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}

func updateTestSecrets(test *testsv3.Test, secretName string, secrets map[string]string) {
	if _, ok := secrets[gitUsernameSecretName]; ok {
		if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.UsernameSecret == nil {
			test.Spec.Content.Repository.UsernameSecret = &testsv3.SecretRef{
				Name: secretName,
				Key:  gitUsernameSecretName,
			}
		}
	} else {
		if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.UsernameSecret != nil &&
			test.Spec.Content.Repository.UsernameSecret.Name == secretName {
			test.Spec.Content.Repository.UsernameSecret = nil
		}
	}

	if _, ok := secrets[gitTokenSecretName]; ok {
		if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.TokenSecret == nil {
			test.Spec.Content.Repository.TokenSecret = &testsv3.SecretRef{
				Name: secretName,
				Key:  gitTokenSecretName,
			}
		}
	} else {
		if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.TokenSecret != nil &&
			test.Spec.Content.Repository.TokenSecret.Name == secretName {
			test.Spec.Content.Repository.TokenSecret = nil
		}
	}
}

func clearTestSecrets(test *testsv3.Test, secretName string) {
	if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.UsernameSecret != nil &&
		test.Spec.Content.Repository.UsernameSecret.Name == secretName {
		test.Spec.Content.Repository.UsernameSecret = nil
	}

	if test.Spec.Content != nil && test.Spec.Content.Repository != nil && test.Spec.Content.Repository.TokenSecret != nil &&
		test.Spec.Content.Repository.TokenSecret.Name == secretName {
		test.Spec.Content.Repository.TokenSecret = nil
	}
}
