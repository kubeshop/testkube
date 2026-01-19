package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func concat(args ...interface{}) []string {
	result := make([]string, 0)
	for _, a := range args {
		switch a := a.(type) {
		case string:
			result = append(result, a)
		case int:
			result = append(result, strconv.Itoa(a))
		case []string:
			result = append(result, a...)
		case []interface{}:
			result = append(result, concat(a...)...)
		}
	}
	return result
}

func Comm(cmd string, args ...interface{}) *exec.Cmd {
	return exec.Command(cmd, concat(args...)...)
}

func Run(c string, args ...interface{}) error {
	sub := Comm(c, args...)
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	return sub.Run()
}

func RunWithRetry(retries int, delay time.Duration, c string, args ...interface{}) (err error) {
	for i := 0; i < retries; i++ {
		err = Run(c, args...)
		if err == nil {
			return nil
		}
		if i+1 < retries {
			nextDelay := time.Duration(i+1) * delay
			fmt.Printf("error, trying again in %s (attempt %d/%d): %s\n", nextDelay.String(), i+2, retries, err.Error())
			time.Sleep(nextDelay)
		}
	}
	return err
}
