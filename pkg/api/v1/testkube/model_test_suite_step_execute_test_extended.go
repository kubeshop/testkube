package testkube

import "fmt"

func (s TestSuiteStepExecuteTest) FullName() string {
	return fmt.Sprintf("run:%s/%s", s.Namespace, s.Name)
}

func (t TestSuiteStepExecuteTest) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}
