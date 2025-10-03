package grpc_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"
)

// generateCertificate creates a CA and a TLS certificate for use when emulating a gRPC server
// with TLS for testing.
func generateCertificate(t *testing.T) (caCert *x509.Certificate, cert *tls.Certificate) {
	t.Helper()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	// Generate CA certificate.
	_, caPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caSerialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatal(err)
	}
	caCertTemplate := x509.Certificate{
		IsCA:         true,
		SerialNumber: caSerialNumber,
		Subject: pkix.Name{
			Organization: []string{"Testkube"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	caDERBytes, err := x509.CreateCertificate(rand.Reader, &caCertTemplate, &caCertTemplate, caPriv.Public(), caPriv)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err = x509.ParseCertificate(caDERBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Generate TLS certificate signed by new CA.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Testkube"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, priv.Public(), caPriv)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	privPemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	newCert, err := tls.X509KeyPair(pemBytes, privPemBytes)
	if err != nil {
		t.Fatal(err)
	}

	return caCert, &newCert
}
