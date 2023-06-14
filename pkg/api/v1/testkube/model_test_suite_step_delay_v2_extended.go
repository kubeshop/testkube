package testkube

import "fmt"

func (s TestSuiteStepDelayV2) FullName() string {
	return fmt.Sprintf("delay %d", s.Duration)
}
