package http

import (
	"net"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
)

const (
	NetDialTimeout       = 5 * time.Second
	TLSHandshakeTimeout  = 5 * time.Second
	DefaultClientTimeout = 10 * time.Second
)

func NewClient(timeout time.Duration) *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: NetDialTimeout,
		}).Dial,
		TLSHandshakeTimeout: TLSHandshakeTimeout,
	}
	c := http.Client{
		Timeout:   DefaultClientTimeout,
		Transport: netTransport,
	}
	if timeout.Seconds() != 0 {
		c.Timeout = timeout
	}
	spew.Dump("=============== TIMEOUT SET TO ===============")
	spew.Dump(c.Timeout)
	spew.Dump("==============================================")
	return &c
}

// NewSSEClient is HTTP client with long timeout to be able to read SSE endpoints
func NewSSEClient() *http.Client {
	return &http.Client{
		Timeout: time.Hour,
	}
}
