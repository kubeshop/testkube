package tls

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"os"
)

// GetTLSConfig returns TLS config based on provided parameters
// - insecure - if true, TLS is disabled
// - skipVerify - if true, server certificate is not verified
// - certFile - path to client certificate
// - keyFile - path to client key
// - caFile - path to root CA
func GetTLSConfig(insecure, skipVerify bool, certFile, keyFile, caFile string) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if !insecure {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		if skipVerify {
			tlsConfig.InsecureSkipVerify = true
		} else {
			if certFile != "" && keyFile != "" {
				cert, err := LoadCertificate(certFile, keyFile)
				if err != nil {
					return nil, errors.Wrap(err, "error loading client certificate")
				}
				tlsConfig.Certificates = []tls.Certificate{*cert}
			}
			if caFile != "" {
				pool, err := LoadRootCAs(caFile)
				if err != nil {
					return nil, errors.Wrap(err, "error loading root CA")
				}
				tlsConfig.RootCAs = pool
			}
		}
	}
	return tlsConfig, nil
}

// LoadCertificate loads client certificate from provided files
// - certFile - path to client certificate
// - keyFile - path to client key
func LoadCertificate(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "error loading client certificate")
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "error parsing client certificate")
	}
	return &cert, nil
}

// LoadRootCAs loads root CA from provided files
// - files - list of files containing root CA
func LoadRootCAs(files ...string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, f := range files {
		rootPEM, err := os.ReadFile(f)
		if err != nil || rootPEM == nil {
			return nil, errors.Wrap(err, "error loading or parsing rootCA file")
		}
		ok := pool.AppendCertsFromPEM(rootPEM)
		if !ok {
			return nil, errors.Wrapf(err, "failed to parse root certificate from %q", f)
		}
	}
	return pool, nil
}
