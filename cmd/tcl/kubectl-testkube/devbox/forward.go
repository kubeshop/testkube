// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/kubeshop/testkube/pkg/ui"
)

func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func ForwardPodPort(config *rest.Config, namespace, podName string, clusterPort, localPort int) error {
	middlewarePort, err := GetFreePort()
	if err != nil {
		return err
	}
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(config.Host, "https://")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &serverURL)
	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", middlewarePort, clusterPort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return err
	}
	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			ui.Fail(errors.Wrap(err, "failed to forward ports"))
		}
		fmt.Println("finish forwarding ports")
	}()

	// Hack to handle Kubernetes Port Forwarding issue.
	// Stream through a different server, to ensure that both connections are fully read, with no broken pipe.
	// @see {@link https://github.com/kubernetes/kubernetes/issues/74551}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return err
	}
	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err == nil {
				go func(conn net.Conn) {
					defer conn.Close()
					open, err := net.Dial("tcp", fmt.Sprintf(":%d", middlewarePort))
					if err != nil {
						return
					}
					defer open.Close()
					var wg sync.WaitGroup
					wg.Add(2)
					go func() {
						io.Copy(open, conn)
						wg.Done()
					}()
					go func() {
						io.Copy(conn, open)
						wg.Done()
					}()
					wg.Wait()

					// Read all before closing
					io.ReadAll(conn)
					io.ReadAll(open)
				}(conn)
			}
		}
	}()

	for range readyChan {
	}
	go func() {
		for {
			http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d", localPort), nil)
			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

func CreateCertificate(cert x509.Certificate) (rcaPEM, rcrtPEM, rkeyPEM []byte, err error) {
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
		return nil, nil, nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

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
		return nil, nil, nil, err
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	return caPEM.Bytes(), certPEM.Bytes(), certPrivKeyPEM.Bytes(), nil
}

func CreateSelfSignedCertificate(tml x509.Certificate) (tls.Certificate, []byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	tml.NotBefore = time.Now()
	tml.NotAfter = time.Now().AddDate(5, 0, 0)
	tml.SerialNumber = big.NewInt(123456)
	tml.BasicConstraintsValid = true
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	tlsCert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}
	return tlsCert, certPem, keyPem, nil
}

func ProxySSL(sourcePort, sslPort int) error {
	tlsCert, _, _, err := CreateSelfSignedCertificate(x509.Certificate{
		IPAddresses: []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1")},
	})
	if err != nil {
		return err
	}
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", sslPort), &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		InsecureSkipVerify: true,
	})
	if err != nil {
		return err
	}
	go func() {
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err == nil {
				go func(conn net.Conn) {
					defer conn.Close()
					open, err := net.Dial("tcp", fmt.Sprintf(":%d", sourcePort))
					if err != nil {
						return
					}
					defer open.Close()
					var wg sync.WaitGroup
					wg.Add(2)
					go func() {
						io.Copy(open, conn)
						wg.Done()
					}()
					go func() {
						io.Copy(conn, open)
						wg.Done()
					}()
					wg.Wait()

					io.ReadAll(conn)
					io.ReadAll(open)
				}(conn)
			}
		}
	}()
	return nil
}
