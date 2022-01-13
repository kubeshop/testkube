package testkube

import "fmt"

func (s TestStepDelay) FullName() string {
	return fmt.Sprintf("delay %dms", s.Duration)
}
