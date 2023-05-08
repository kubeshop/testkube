package client

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/kubeshop/testkube/pkg/http"
)

func NewOrganizationsClient(token string) *Organizations {
	return &Organizations{
		BaseUrl: "https://api.testkube.io/organizations",
		Client:  http.NewClient(),
		token:   token,
	}
}

type OrganizationResponse struct {
	Elements []Organization `json:"elements"`
}
type Organization struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type Organizations struct {
	BaseUrl string
	Client  http.HttpClient
	token   string
}

func (o *Organizations) Get() ([]Organization, error) {
	r, err := nethttp.NewRequest("GET", o.BaseUrl+"", nil)
	r.Header.Add("Authorization", "Bearer "+o.token)
	if err != nil {
		return nil, err
	}
	resp, err := o.Client.Do(r)
	if err != nil {
		return nil, err
	}

	var orgsResponse OrganizationResponse
	err = json.NewDecoder(resp.Body).Decode(&orgsResponse)
	return orgsResponse.Elements, err
}
