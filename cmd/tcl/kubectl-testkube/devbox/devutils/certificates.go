// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

type CertificateSet struct {
	CaPEM  []byte
	CrtPEM []byte
	KeyPEM []byte
}

func CreateCertificate(cert x509.Certificate) (result CertificateSet, err error) {
	// Build CA
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(11111),
		Subject: pkix.Name{
			Organization:  []string{"Kubeshop"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Wilmington"},
			StreetAddress: []string{"Orange St"},
			PostalCode:    []string{"19801"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return result, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return result, err
	}
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey)})

	// Build the direct certificate
	cert.NotBefore = ca.NotBefore
	cert.NotAfter = ca.NotAfter
	cert.SerialNumber = big.NewInt(11111)
	cert.Subject = ca.Subject
	cert.SubjectKeyId = []byte{1, 2, 3, 4, 6}
	cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	cert.KeyUsage = x509.KeyUsageDigitalSignature

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return result, err
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return result, err
	}
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey)})

	result.CaPEM = caPEM.Bytes()
	result.CrtPEM = certPEM.Bytes()
	result.KeyPEM = certPrivKeyPEM.Bytes()
	return result, nil
}
