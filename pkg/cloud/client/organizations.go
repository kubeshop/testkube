package client

import (
	"github.com/kubeshop/testkube/pkg/http"
)

func NewOrganizationsClient(rootDomain, token string) *OrganizationsClient {
	return &OrganizationsClient{
		RESTClient: RESTClient[Organization]{
			BaseUrl: "https://api." + rootDomain,
			Path:    "/organizations",
			Client:  http.NewClient(),
			Token:   token,
		},
	}
}

type Organization struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type OrganizationsClient struct {
	RESTClient[Organization]
}
