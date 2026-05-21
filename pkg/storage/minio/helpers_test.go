package minio

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTLSOptions(t *testing.T) {
	t.Run("ssl disabled returns no options", func(t *testing.T) {
		opts := GetTLSOptions(false, false, "", "", "")
		assert.Empty(t, opts)
	})

	t.Run("ssl enabled with skipVerify returns Insecure option", func(t *testing.T) {
		opts := GetTLSOptions(true, true, "", "", "")
		assert.Len(t, opts, 1)

		// Apply option to verify it sets InsecureSkipVerify
		connecter := &Connecter{}
		err := opts[0](connecter)
		require.NoError(t, err)
		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.True(t, connecter.TlsConfig.InsecureSkipVerify)
	})

	t.Run("ssl enabled without skipVerify and no certs returns Secure option", func(t *testing.T) {
		// SSL enabled without certificates should still enable HTTPS connections
		opts := GetTLSOptions(true, false, "", "", "")
		assert.Len(t, opts, 1)

		// Apply option to verify it sets Ssl to true
		connecter := &Connecter{}
		err := opts[0](connecter)
		require.NoError(t, err)
		assert.True(t, connecter.Ssl)
	})

	t.Run("ssl enabled with only CA file returns Secure and RootCAs options", func(t *testing.T) {
		// Create a temporary CA certificate file for testing
		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "ca.pem")

		// Generate a simple PEM certificate for testing
		certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

		err := os.WriteFile(caFile, certPEM, 0644)
		require.NoError(t, err)

		opts := GetTLSOptions(true, false, "", "", caFile)
		assert.Len(t, opts, 2)

		// Apply options to verify they set Ssl and RootCAs
		connecter := &Connecter{}
		for _, opt := range opts {
			err = opt(connecter)
			require.NoError(t, err)
		}
		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.NotNil(t, connecter.TlsConfig.RootCAs)
	})

	t.Run("ssl enabled with client certs returns Secure and ClientCert options", func(t *testing.T) {
		// Create temporary certificate files for testing
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")

		// Generate a simple self-signed certificate and key for testing
		certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

		keyPEM := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

		err := os.WriteFile(certFile, certPEM, 0644)
		require.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0600)
		require.NoError(t, err)

		opts := GetTLSOptions(true, false, certFile, keyFile, "")
		assert.Len(t, opts, 2)

		// Apply options to verify they set Ssl and client certificate
		connecter := &Connecter{}
		for _, opt := range opts {
			err = opt(connecter)
			require.NoError(t, err)
		}
		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.Len(t, connecter.TlsConfig.Certificates, 1)
	})

	t.Run("ssl enabled with client certs and CA returns Secure, ClientCert, and RootCAs options", func(t *testing.T) {
		// Create temporary certificate files for testing
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")
		caFile := filepath.Join(tmpDir, "ca.pem")

		// Generate certificates
		certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

		keyPEM := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

		err := os.WriteFile(certFile, certPEM, 0644)
		require.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0600)
		require.NoError(t, err)
		err = os.WriteFile(caFile, certPEM, 0644)
		require.NoError(t, err)

		opts := GetTLSOptions(true, false, certFile, keyFile, caFile)
		assert.Len(t, opts, 3)

		// Apply options to verify both are configured
		connecter := &Connecter{}
		for _, opt := range opts {
			err = opt(connecter)
			require.NoError(t, err)
		}
		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.Len(t, connecter.TlsConfig.Certificates, 1)
		assert.NotNil(t, connecter.TlsConfig.RootCAs)
	})

	t.Run("ssl enabled with only certFile returns Secure option only", func(t *testing.T) {
		// If only certFile is provided without keyFile, should not attempt to load client cert
		// but should still enable SSL
		opts := GetTLSOptions(true, false, "/path/to/cert", "", "")
		assert.Len(t, opts, 1)

		connecter := &Connecter{}
		err := opts[0](connecter)
		require.NoError(t, err)
		assert.True(t, connecter.Ssl)
		assert.Nil(t, connecter.TlsConfig)
	})

	t.Run("ssl enabled with only keyFile returns Secure option only", func(t *testing.T) {
		// If only keyFile is provided without certFile, should not attempt to load client cert
		// but should still enable SSL
		opts := GetTLSOptions(true, false, "", "/path/to/key", "")
		assert.Len(t, opts, 1)

		connecter := &Connecter{}
		err := opts[0](connecter)
		require.NoError(t, err)
		assert.True(t, connecter.Ssl)
		assert.Nil(t, connecter.TlsConfig)
	})
}

func TestClientCertOption(t *testing.T) {
	t.Run("returns error when cert file does not exist", func(t *testing.T) {
		opt := ClientCert("/nonexistent/cert.pem", "/nonexistent/key.pem")
		connecter := &Connecter{}
		err := opt(connecter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error loading client certificate")
	})

	t.Run("returns error when cert path is empty", func(t *testing.T) {
		opt := ClientCert("", "")
		connecter := &Connecter{}
		err := opt(connecter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error loading client certificate")
	})
}

func TestRootCAsOption(t *testing.T) {
	t.Run("returns error when CA file does not exist", func(t *testing.T) {
		opt := RootCAs("/nonexistent/ca.pem")
		connecter := &Connecter{}
		err := opt(connecter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error loading or parsing rootCA file")
	})

	t.Run("initializes TlsConfig with RootCAs pool", func(t *testing.T) {
		// Create a temporary CA certificate file for testing
		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "ca.pem")

		certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

		err := os.WriteFile(caFile, certPEM, 0644)
		require.NoError(t, err)

		opt := RootCAs(caFile)
		connecter := &Connecter{}
		err = opt(connecter)
		require.NoError(t, err)

		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.NotNil(t, connecter.TlsConfig.RootCAs)
		assert.Equal(t, uint16(tls.VersionTLS12), connecter.TlsConfig.MinVersion)
	})
}

func TestInsecureOption(t *testing.T) {
	t.Run("sets InsecureSkipVerify to true", func(t *testing.T) {
		opt := Insecure()
		connecter := &Connecter{}
		err := opt(connecter)
		require.NoError(t, err)

		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.True(t, connecter.TlsConfig.InsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), connecter.TlsConfig.MinVersion)
	})

	t.Run("preserves existing TlsConfig", func(t *testing.T) {
		connecter := &Connecter{
			TlsConfig: &tls.Config{
				RootCAs: x509.NewCertPool(),
			},
		}

		opt := Insecure()
		err := opt(connecter)
		require.NoError(t, err)

		assert.True(t, connecter.Ssl)
		assert.NotNil(t, connecter.TlsConfig)
		assert.True(t, connecter.TlsConfig.InsecureSkipVerify)
		assert.NotNil(t, connecter.TlsConfig.RootCAs, "Should preserve existing RootCAs")
	})
}
