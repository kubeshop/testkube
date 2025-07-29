// Package testworkflowexecutor is a package that handles test workflow scheduling and execution
package testworkflowexecutor

import "fmt"

// DuplicateTargetError happens when an execution uses two templates each with a target defined.
type DuplicateTargetError struct {
	Template1 string
	Template2 string
}

func (e DuplicateTargetError) Error() string {
	return fmt.Sprintf("cannot define target within multiple templates: %s, %s", e.Template1, e.Template2)
}
