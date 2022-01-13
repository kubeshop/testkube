package testkube

import "fmt"

func (s TestStepExecuteScript) FullName() string {
	return fmt.Sprintf("run script: %s/%s", s.Namespace, s.Name)
}
