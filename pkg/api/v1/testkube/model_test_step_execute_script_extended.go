package testkube

import "fmt"

func (s TestStepExecuteScript) FullName() string {
	return fmt.Sprintf("run:%s/%s", s.Namespace, s.Name)
}

func (t TestStepExecuteScript) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}
