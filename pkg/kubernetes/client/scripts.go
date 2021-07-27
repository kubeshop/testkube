package client

import (
	"context"

	scriptsAPI "github.com/kubeshop/kubetest/internal/app/operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewScripts(client client.Client) Scripts {
	return Scripts{
		Client: client,
	}
}

type Scripts struct {
	Client client.Client
}

func (s Scripts) List(namespace string) (result *scriptsAPI.ScriptList, err error) {
	list := &scriptsAPI.ScriptList{}
	err = s.Client.List(context.Background(), list, &client.ListOptions{Namespace: namespace})
	return list, err
}

func (s Scripts) Create(deployment *scriptsAPI.Script) (sdep *scriptsAPI.Script, err error) {
	err = s.Client.Create(context.Background(), deployment)
	return deployment, err
}
