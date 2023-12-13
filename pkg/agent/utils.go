package agent

import (
	"crypto/tls"
	tlsutil "github.com/kubeshop/testkube/internal/tls"
	"github.com/pkg/errors"
)

func buildTLSConfig(config Config) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if !config.Insecure {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		if config.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
		} else {
			if config.CertFile != "" && config.KeyFile != "" {
				cert, err := tlsutil.LoadCertificate(config.CertFile, config.KeyFile)
				if err != nil {
					return nil, errors.Wrap(err, "error loading client certificate")
				}
				tlsConfig.Certificates = []tls.Certificate{*cert}
			}
			if config.CAFile != "" {
				pool, err := tlsutil.LoadRootCAs(config.CAFile)
				if err != nil {
					return nil, errors.Wrap(err, "error loading root CA")
				}
				tlsConfig.RootCAs = pool
			}
		}
	}
	return tlsConfig, nil
}
