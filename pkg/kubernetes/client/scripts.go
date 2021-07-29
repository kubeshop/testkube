package client

import (
	"context"

	scriptsAPI "github.com/kubeshop/kubetest/internal/app/operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewScriptsKubeAPI(client client.Client) ScriptsKubeAPI {
	return ScriptsKubeAPI{
		Client: client,
	}
}

type ScriptsKubeAPI struct {
	Client client.Client
}

func (s ScriptsKubeAPI) List(namespace string) (*scriptsAPI.ScriptList, error) {
	list := &scriptsAPI.ScriptList{}
	err := s.Client.List(context.Background(), list, &client.ListOptions{Namespace: namespace})
	return list, err
}

func (s ScriptsKubeAPI) Create(scripts *scriptsAPI.Script) (*scriptsAPI.Script, error) {
	err := s.Client.Create(context.Background(), scripts)
	return scripts, err
}
