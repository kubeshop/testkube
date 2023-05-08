package client

import (
	"github.com/kubeshop/testkube/pkg/http"
)

func NewEnvironmentsClient(token string) *EnvironmentsClient {
	return &EnvironmentsClient{
		RESTClient: RESTClient[Environment]{
			BaseUrl: "https://api.testkube.io",
			Path:    "/organizations",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type Environment struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type EnvironmentsClient struct {
	RESTClient[Environment]
}
