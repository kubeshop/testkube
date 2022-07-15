package testkube

func NewGitRepository(uri, branch, commit string) *Repository {
	return &Repository{
		Type_:  "git",
		Branch: branch,
		Commit: commit,
		Uri:    uri,
	}
}

func NewAuthGitRepository(uri, branch, commit, user, token string) *Repository {
	return &Repository{
		Type_:    "git",
		Branch:   branch,
		Commit:   commit,
		Uri:      uri,
		Username: user,
		Token:    token,
	}
}

func (r *Repository) WithPath(path string) *Repository {
	r.Path = path
	return r
}
