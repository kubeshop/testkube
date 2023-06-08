package http

import (
	"net"
	"net/http"
	"time"
)

const (
	NetDialTimeout      = 30 * time.Second
	TLSHandshakeTimeout = 30 * time.Second
	ClientTimeout       = 5 * time.Minute
)

func NewClient() *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: NetDialTimeout,
		}).Dial,
		TLSHandshakeTimeout: TLSHandshakeTimeout,
		Proxy:               http.ProxyFromEnvironment,
	}
	return &http.Client{
		Timeout:   ClientTimeout,
		Transport: netTransport,
	}
}

// NewSSEClient is HTTP client with long timeout to be able to read SSE endpoints
func NewSSEClient() *http.Client {
	return &http.Client{
		Timeout: time.Hour,
	}
}
