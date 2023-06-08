package testkube

import "fmt"

func (s TestSuiteStepDelay) FullName() string {
	return fmt.Sprintf("delay %s", s.Duration)
}
