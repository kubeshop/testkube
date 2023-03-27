package testkube

// IsEmpty check if update is empty
func (r *RepositoryUpdate) IsEmpty() bool {
	var stringFields = []*string{r.Type_, r.Uri, r.Branch, r.Commit, r.Path,
		r.Username, r.Token, r.CertificateSecret, r.WorkingDir, r.AuthType}
	var secretRefs = []**SecretRef{r.UsernameSecret, r.TokenSecret}

	for _, field := range stringFields {
		if field != nil {
			return false
		}
	}

	for _, field := range secretRefs {
		if field != nil {
			return false
		}
	}

	return true
}
