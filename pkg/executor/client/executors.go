package client

import (
	"sync"

	executorscr "github.com/kubeshop/kubtest-operator/client/executors"
)

func NewExecutors(client *executorscr.ExecutorsClient) Executors {
	return Executors{
		ExecutorsCRClient: client,
	}
}

// Executors represents available HTTP clients for executors registered in Kubernetes API
type Executors struct {
	ExecutorsCRClient *executorscr.ExecutorsClient
	Namespace         string
	Clients           sync.Map
}

// Get gets executor based on type with a basic map.Sync cache
// TODO add TTL someday - to handle changes
// TODO there is no handling of CR change
func (p *Executors) Get(scriptType string) (executorClient HTTPExecutorClient, err error) {

	cached, exists := p.Clients.Load(scriptType)
	if !exists {

		executorCR, err := p.ExecutorsCRClient.GetByType(scriptType)
		if err != nil {
			return executorClient, err
		}

		executor := NewHTTPExecutorClient(Config{
			URI: executorCR.Spec.URI,
		})

		p.Clients.Store(scriptType, executor)
		cached = executor
	}

	executorClient = cached.(HTTPExecutorClient)
	return
}
