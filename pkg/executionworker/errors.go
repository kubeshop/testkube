package executionworker

import "errors"

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrPodIpNotAssigned = errors.New("selected pod does not have IP assigned")
)
