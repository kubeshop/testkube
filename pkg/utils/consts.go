package utils

import "time"

const (
	// DefaultDockerRegistry is the default registry used when no registry is specified in the image name.
	DefaultDockerRegistry = "https://index.docker.io/v1/"
	DefaultRetryDelay     = time.Second * 3
)
