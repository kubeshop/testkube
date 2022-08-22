package event

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type Listener interface {
	Notify(event testkube.TestkubeEvent)
}
