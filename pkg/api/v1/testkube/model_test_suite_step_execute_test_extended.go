package testkube

import "fmt"

func (s TestSuiteStepExecuteTest) FullName() string {
	return fmt.Sprintf("run:%s/%s", s.Namespace, s.Name)
}

func (s TestSuiteStepExecuteTest) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      s.Name,
		Namespace: s.Namespace,
	}
}
