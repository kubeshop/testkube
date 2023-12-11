package testkube

// IsEmpty check if request is empty
func (a *ArtifactUpdateRequest) IsEmpty() bool {
	if a.StorageClassName != nil || a.VolumeMountPath != nil || a.Dirs != nil || a.Masks != nil ||
		a.StorageBucket != nil || a.OmitFolderPerExecution != nil || a.SharedBetweenPods != nil {
		return false
	}

	return true
}
