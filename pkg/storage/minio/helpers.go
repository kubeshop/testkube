package minio

func GetTLSOptions(ssl, skipVerify bool, certFile, keyFile, caFile string) []Option {
	var opts []Option
	if ssl {
		if skipVerify {
			opts = append(opts, Insecure())
		} else {
			opts = append(opts, ClientCert(certFile, keyFile))
			if caFile != "" {
				opts = append(opts, RootCAs(caFile))
			}
		}
	}
	return opts
}
