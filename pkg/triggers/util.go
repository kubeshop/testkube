package triggers

import (
	"time"

	core_v1 "k8s.io/api/core/v1"
)

func findContainer(containers []core_v1.Container, target string) *core_v1.Container {
	for _, c := range containers {
		if c.Name == target {
			return &c
		}
	}
	return nil
}

func inPast(t1, t2 time.Time) bool {
	return t1.Before(t2)
}
