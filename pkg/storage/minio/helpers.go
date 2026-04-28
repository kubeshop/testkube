package minio

func GetTLSOptions(ssl, skipVerify bool, certFile, keyFile, caFile string) []Option {
	var opts []Option
	if ssl {
		if skipVerify {
			opts = append(opts, Insecure())
		} else {
			// Only load client certificates if both certFile and keyFile are provided
			if certFile != "" && keyFile != "" {
				opts = append(opts, ClientCert(certFile, keyFile))
			}
			if caFile != "" {
				opts = append(opts, RootCAs(caFile))
			}
		}
	}
	return opts
}
