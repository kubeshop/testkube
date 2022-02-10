package testkube

import "fmt"

func (s TestSuiteStepExecuteScript) FullName() string {
	return fmt.Sprintf("run:%s/%s", s.Namespace, s.Name)
}

func (t TestSuiteStepExecuteScript) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}
