package mock

import (
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
)

type TestSuiteClient struct {
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

func (c TestSuiteClient) Create(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.CreateFn == nil {
		panic("not implemented")
	}
	return c.CreateFn(testsuite)
}

func (c TestSuiteClient) Get(name string) (*testsuitesv2.TestSuite, error) {
	if c.GetFn == nil {
		panic("not implemented")
	}
	return c.GetFn(name)
}
func (c TestSuiteClient) GetSecretTestSuiteVars(testSuiteName, secretUUID string) (map[string]string, error) {
	if c.GetSecretTestSuiteVarsFn == nil {
		panic("not implemented")
	}
	return c.GetSecretTestSuiteVarsFn(testSuiteName, secretUUID)
}

func (c TestSuiteClient) GetCurrentSecretUUID(testSuiteName string) (string, error) {
	if c.GetCurrentSecretUUIDFn == nil {
		panic("not implemented")
	}
	return c.GetCurrentSecretUUIDFn(testSuiteName)
}

func (c TestSuiteClient) List(selector string) (*testsuitesv2.TestSuiteList, error) {
	if c.ListFn == nil {
		panic("not implemented")
	}
	return c.ListFn(selector)
}

func (c TestSuiteClient) ListLabels() (map[string][]string, error) {
	if c.ListLabelsFn == nil {
		panic("not implemented")
	}
	return c.ListLabelsFn()
}

func (c TestSuiteClient) Update(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.UpdateFn == nil {
		panic("not implemented")
	}
	return c.UpdateFn(testsuite)
}

func (c TestSuiteClient) Delete(name string) error {
	if c.DeleteFn == nil {
		panic("not implemented")
	}
	return c.DeleteFn(name)
}

func (c TestSuiteClient) DeleteAll() error {
	if c.DeleteAllFn == nil {
		panic("not implemented")
	}
	return c.DeleteAllFn()
}

func (c TestSuiteClient) DeleteByLabels(selector string) error {
	if c.DeleteByLabelsFn == nil {
		panic("not implemented")
	}
	return c.DeleteByLabelsFn(selector)
}
