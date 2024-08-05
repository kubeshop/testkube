package controlplane

type GetEnvironmentRequest struct {
	Token string
}

type GetEnvironmentResponse struct {
	Id   string
	Name string
	Slug string

	OrganizationId   string
	OrganizationSlug string
	OrganizationName string
}
