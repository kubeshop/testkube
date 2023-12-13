package tls

import (
	"crypto/tls"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestGetTLSConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		insecure         bool
		skipVerify       bool
		certFile         string
		keyFile          string
		caFile           string
		wantError        bool
		wantErrorMessage string
		checkTLS         func(*tls.Config) bool
	}{
		{
			name:      "Insecure mode",
			insecure:  true,
			wantError: false,
			checkTLS: func(cfg *tls.Config) bool {
				return cfg == nil
			},
		},
		{
			name:       "Skip verify",
			skipVerify: true,
			wantError:  false,
			checkTLS: func(cfg *tls.Config) bool {
				return cfg.InsecureSkipVerify
			},
		},
		{
			name:      "Valid certificate and key",
			certFile:  testDataPath("client.crt"),
			keyFile:   testDataPath("client.key"),
			wantError: false,
			checkTLS: func(cfg *tls.Config) bool {
				return len(cfg.Certificates) == 1
			},
		},
		{
			name:             "Invalid certificate path",
			certFile:         testDataPath("invalid.crt"),
			keyFile:          testDataPath("client.key"),
			wantError:        true,
			wantErrorMessage: "error loading client certificate: open testdata/invalid.crt: no such file or directory",
		},
		{
			name:      "Valid CA file",
			caFile:    testDataPath("ca.crt"),
			wantError: false,
			checkTLS: func(cfg *tls.Config) bool {
				return cfg.RootCAs != nil
			},
		},
		{
			name:      "Invalid CA file path",
			caFile:    testDataPath("invalid-ca.crt"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := GetTLSConfig(tt.insecure, tt.skipVerify, tt.certFile, tt.keyFile, tt.caFile)

			if tt.wantError {
				assert.ErrorContains(t, err, tt.wantErrorMessage)
			} else {
				if err != nil {
					t.Fatalf("GetTLSConfig() unexpected error: %v", err)
				}
				if tt.checkTLS != nil {
					assert.True(t, tt.checkTLS(config))
				}
			}
		})
	}
}

func TestLoadCertificate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		certFile         string
		keyFile          string
		wantError        bool
		wantErrorMessage string
	}{
		{
			name:      "Valid certificate and key",
			certFile:  testDataPath("client.crt"),
			keyFile:   testDataPath("client.key"),
			wantError: false,
		},
		{
			name:             "Invalid certificate file path",
			certFile:         testDataPath("invalid.crt"),
			keyFile:          testDataPath("client.key"),
			wantError:        true,
			wantErrorMessage: "error loading client certificate: open testdata/invalid.crt: no such file or directory",
		},
		// Add more test cases here
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cert, err := LoadCertificate(tt.certFile, tt.keyFile)
			if tt.wantError {
				assert.ErrorContains(t, err, tt.wantErrorMessage)
			} else {
				if err != nil {
					t.Fatalf("LoadCertificate() unexpected error: %v", err)
				}
				assert.NotNil(t, cert)
			}
		})
	}
}

func TestLoadRootCAs(t *testing.T) {
	tests := []struct {
		name             string
		files            []string
		wantError        bool
		wantErrorMessage string
	}{
		{
			name:      "Valid root CA file",
			files:     []string{testDataPath("ca.crt")},
			wantError: false,
		},
		{
			name:             "Invalid root CA file path",
			files:            []string{testDataPath("invalid-ca.crt")},
			wantError:        true,
			wantErrorMessage: "error loading or parsing rootCA file",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool, err := LoadRootCAs(tt.files...)
			if tt.wantError {
				assert.ErrorContains(t, err, tt.wantErrorMessage)
			} else {
				if err != nil {
					t.Fatalf("LoadRootCAs() unexpected error: %v", err)
				}
				assert.NotNil(t, pool)
			}
		})
	}
}

// testDataPath is a helper function to get the absolute path
func testDataPath(file string) string {
	return filepath.Join("testdata", file)
}
