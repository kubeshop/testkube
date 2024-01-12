package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const (
	NetDialTimeout      = 30 * time.Second
	TLSHandshakeTimeout = 30 * time.Second
	ClientTimeout       = 5 * time.Minute
)

func NewClient(insecure ...bool) *http.Client {
	var tlsConfig *tls.Config
	if len(insecure) == 1 && insecure[0] {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: NetDialTimeout,
		}).Dial,
		TLSHandshakeTimeout: TLSHandshakeTimeout,
		Proxy:               http.ProxyFromEnvironment,
		TLSClientConfig:     tlsConfig,
	}
	return &http.Client{
		Timeout:   ClientTimeout,
		Transport: netTransport,
	}
}

// NewSSEClient is HTTP client with long timeout to be able to read SSE endpoints
func NewSSEClient(insecure ...bool) *http.Client {
	var netTransport *http.Transport
	netTransport = http.DefaultTransport.(*http.Transport)
	if len(insecure) == 1 && insecure[0] {
		netTransport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &http.Client{
		Timeout:   time.Hour,
		Transport: netTransport,
	}
}
