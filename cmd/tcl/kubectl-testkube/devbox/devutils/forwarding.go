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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
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
)

func GetFreePort() int {
	var a *net.TCPAddr
	var err error
	if a, err = net.ResolveTCPAddr("tcp", ":0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port
		}
	}
	panic(err)
}

// TODO: Support context
func ForwardPod(config *rest.Config, namespace, podName string, clusterPort, localPort int, ping bool) error {
	middlewarePort := GetFreePort()
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimPrefix(strings.TrimPrefix(config.Host, "http://"), "https://")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &serverURL)
	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", middlewarePort, clusterPort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return err
	}
	go func() {
		for {
			if err = forwarder.ForwardPorts(); err != nil {
				fmt.Println(errors.Wrap(err, "warn: forwarder: closed"))
				time.Sleep(50 * time.Millisecond)
				readyChan = make(chan struct{}, 1)
				forwarder, err = portforward.New(dialer, []string{fmt.Sprintf("%d:%d", middlewarePort, clusterPort)}, stopChan, readyChan, out, errOut)
				go func(readyChan chan struct{}) {
					<-readyChan
					fmt.Println("forwarder: reconnected")
				}(readyChan)
			}
		}
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
					open, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", fmt.Sprintf(":%d", middlewarePort))
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

	if ping {
		go func() {
			for {
				http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d", localPort), nil)
				time.Sleep(4 * time.Second)
			}
		}()
	}

	return nil
}

func ProxySSL(sourcePort, sslPort int) error {
	set, err := CreateCertificate(x509.Certificate{
		IPAddresses: []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1"), net.IPv6loopback},
	})
	if err != nil {
		return err
	}
	crt, err := tls.X509KeyPair(set.CrtPEM, set.KeyPEM)
	if err != nil {
		return err
	}
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", sslPort), &tls.Config{
		Certificates:       []tls.Certificate{crt},
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
					open, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", fmt.Sprintf(":%d", sourcePort))
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
				}(conn)
			}
		}
	}()
	return nil
}
