package registry

import "errors"

var (
	ErrResourceNotFound = errors.New("resource not found agent")
	ErrPodIpNotAssigned = errors.New("selected pod does not have IP assigned")
)
