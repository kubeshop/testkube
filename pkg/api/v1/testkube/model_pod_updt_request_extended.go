package testkube

// IsEmpty check if request is empty
func (p *PodUpdateRequest) IsEmpty() bool {
	if p.Resources != nil {
		if (*p.Resources).Requests != nil && ((*p.Resources).Requests.Cpu != nil || (*p.Resources).Requests.Memory != nil) {
			return false
		}

		if (*p.Resources).Limits != nil && ((*p.Resources).Limits.Cpu != nil || (*p.Resources).Limits.Memory != nil) {
			return false
		}
	}

	if p.PodTemplate != nil || p.PodTemplateReference != nil {
		return false
	}

	return true
}
