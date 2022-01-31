package testkube

func NewGitRepository(uri, branch string) *Repository {
	return &Repository{
		Type_:  "git",
		Branch: "main",
		Uri:    uri,
	}
}

func NewAuthGitRepository(uri, branch, user, token string) *Repository {
	return &Repository{
		Type_:    "git",
		Branch:   branch,
		Uri:      uri,
		Username: user,
		Token:    token,
	}
}

func (r *Repository) WithPath(path string) *Repository {
	r.Path = path
	return r
}
