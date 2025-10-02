package tests

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testsv1 "github.com/kubeshop/testkube/api/tests/v1"
	"github.com/kubeshop/testkube/pkg/utils"
)

func NewClient(client client.Client, namespace string) *TestsClient {
	return &TestsClient{
		Client:    client,
		Namespace: namespace,
	}
}

type TestsClient struct {
	Client    client.Client
	Namespace string
}

func (s TestsClient) List(tags []string) (*testsv1.TestList, error) {
	list := &testsv1.TestList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if len(tags) == 0 {
		return list, err
	}

	toReturn := &testsv1.TestList{}
	for _, test := range list.Items {
		hasTags := false
		for _, tag := range tags {
			if utils.ContainsTag(test.Spec.Tags, tag) {
				hasTags = true
			} else {
				hasTags = false
			}

		}
		if hasTags {
			toReturn.Items = append(toReturn.Items, test)

		}
	}
	return toReturn, nil
}

func (s TestsClient) ListTags() ([]string, error) {
	tags := []string{}
	list := &testsv1.TestList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return tags, err
	}

	for _, test := range list.Items {
		tags = append(tags, test.Spec.Tags...)
	}

	tags = utils.RemoveDuplicates(tags)

	return tags, nil
}

func (s TestsClient) Get(name string) (*testsv1.Test, error) {
	test := &testsv1.Test{}
	err := s.Client.Get(context.Background(), client.ObjectKey{Namespace: s.Namespace, Name: name}, test)
	return test, err
}

func (s TestsClient) Create(test *testsv1.Test) (*testsv1.Test, error) {
	err := s.Client.Create(context.Background(), test)
	return test, err
}

func (s TestsClient) Update(test *testsv1.Test) (*testsv1.Test, error) {
	err := s.Client.Update(context.Background(), test)
	return test, err
}

func (s TestsClient) Delete(name string) error {
	Test, err := s.Get(name)
	if err != nil {
		return err
	}

	err = s.Client.Delete(context.Background(), Test)
	return err
}

func (s TestsClient) DeleteAll() error {
	u := &unstructured.Unstructured{}
	u.SetKind("Test")
	u.SetAPIVersion("tests.testkube.io/v1")
	err := s.Client.DeleteAllOf(context.Background(), u, client.InNamespace(s.Namespace))
	return err
}
