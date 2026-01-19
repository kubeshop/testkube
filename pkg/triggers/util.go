package triggers

import (
	"time"
)

func inPast(t1, t2 time.Time) bool {
	return t1.Before(t2)
}
