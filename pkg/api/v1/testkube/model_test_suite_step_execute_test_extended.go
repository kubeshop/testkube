package testkube

import "fmt"

func (s TestSuiteStepExecuteTest) FullName() string {
	return fmt.Sprintf("run:%s", s.Name)
}

func (s TestSuiteStepExecuteTest) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name: s.Name,
	}
}
