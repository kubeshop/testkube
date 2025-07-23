package testworkflowexecutor

import "fmt"

type DuplicateTargetError struct {
	Template1 string
	Template2 string
}

func (e DuplicateTargetError) Error() string {
	return fmt.Sprintf("cannot define target within multiple templates: %s, %s", e.Template1, e.Template2)
}
