package client

import (
	"github.com/kubeshop/testkube/pkg/http"
)

func NewOrganizationsClient(url, token string, insecure ...bool) *OrganizationsClient {
	return &OrganizationsClient{
		RESTClient: RESTClient[Organization, Organization]{
			BaseUrl: url,
			Path:    "/organizations",
			Client:  http.NewClient(insecure...),
			Token:   token,
		},
	}
}

type Organization struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type OrganizationsClient struct {
	RESTClient[Organization, Organization]
}
