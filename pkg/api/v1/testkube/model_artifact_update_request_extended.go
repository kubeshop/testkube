package testkube

func (a *ArtifactUpdateRequest) IsEmpty() bool {
	if a.StorageClassName != nil || a.VolumeMountPath != nil || a.Dirs != nil {
		return false
	}

	return true
}
