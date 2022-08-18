package mock

import (
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
)

type TestSuitesClient struct {
	CreateFn                 func(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error)
	GetFn                    func(name string) (*testsuitesv2.TestSuite, error)
	GetSecretTestSuiteVarsFn func(testSuiteName, secretUUID string) (map[string]string, error)
	GetCurrentSecretUUIDFn   func(testSuiteName string) (string, error)
	ListFn                   func(selector string) (*testsuitesv2.TestSuiteList, error)
	ListLabelsFn             func() (map[string][]string, error)
	UpdateFn                 func(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error)
	DeleteFn                 func(name string) error
	DeleteAllFn              func() error
	DeleteByLabelsFn         func(selector string) error
}

func (c TestSuitesClient) Create(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.CreateFn == nil {
		panic("not implemented")
	}
	return c.CreateFn(testsuite)
}

func (c TestSuitesClient) Get(name string) (*testsuitesv2.TestSuite, error) {
	if c.GetFn == nil {
		panic("not implemented")
	}
	return c.GetFn(name)
}
func (c TestSuitesClient) GetSecretTestSuiteVars(testSuiteName, secretUUID string) (map[string]string, error) {
	if c.GetSecretTestSuiteVarsFn == nil {
		panic("not implemented")
	}
	return c.GetSecretTestSuiteVarsFn(testSuiteName, secretUUID)
}

func (c TestSuitesClient) GetCurrentSecretUUID(testSuiteName string) (string, error) {
	if c.GetCurrentSecretUUIDFn == nil {
		panic("not implemented")
	}
	return c.GetCurrentSecretUUIDFn(testSuiteName)
}

func (c TestSuitesClient) List(selector string) (*testsuitesv2.TestSuiteList, error) {
	if c.ListFn == nil {
		panic("not implemented")
	}
	return c.ListFn(selector)
}

func (c TestSuitesClient) ListLabels() (map[string][]string, error) {
	if c.ListLabelsFn == nil {
		panic("not implemented")
	}
	return c.ListLabelsFn()
}

func (c TestSuitesClient) Update(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.UpdateFn == nil {
		panic("not implemented")
	}
	return c.UpdateFn(testsuite)
}

func (c TestSuitesClient) Delete(name string) error {
	if c.DeleteFn == nil {
		panic("not implemented")
	}
	return c.DeleteFn(name)
}

func (c TestSuitesClient) DeleteAll() error {
	if c.DeleteAllFn == nil {
		panic("not implemented")
	}
	return c.DeleteAllFn()
}

func (c TestSuitesClient) DeleteByLabels(selector string) error {
	if c.DeleteByLabelsFn == nil {
		panic("not implemented")
	}
	return c.DeleteByLabelsFn(selector)
}
