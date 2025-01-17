package runner

import (
	"time"
)

func retry(count int, delayBase time.Duration, fn func() error) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}
