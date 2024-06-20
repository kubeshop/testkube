package testkube

func (s *ServerInfo) SecretConfig() *SecretConfig {
	if s.Secret != nil {
		return s.Secret
	}
	// CLI fallback for older Agents
	return &SecretConfig{
		Prefix:     "",
		List:       s.EnableSecretEndpoint,
		ListAll:    s.EnableSecretEndpoint,
		Create:     s.EnableSecretEndpoint && !s.DisableSecretCreation,
		Modify:     s.EnableSecretEndpoint && !s.DisableSecretCreation,
		Delete:     s.EnableSecretEndpoint && !s.DisableSecretCreation,
		AutoCreate: !s.DisableSecretCreation,
	}
}
