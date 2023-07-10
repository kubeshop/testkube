package testkube

import "fmt"

func (s TestSuiteStepExecuteTestV2) FullName() string {
	return fmt.Sprintf("run:%s", s.Name)
}

func (s TestSuiteStepExecuteTestV2) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name: s.Name,
	}
}
