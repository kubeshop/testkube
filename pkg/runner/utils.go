package runner

import (
	"time"
)

func retry(count int, delayBase time.Duration, fn func(retryCount int) error) (err error) {
	for i := 0; i < count; i++ {
		err = fn(i)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}
