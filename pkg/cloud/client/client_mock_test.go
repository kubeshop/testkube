package client

import (
	"bytes"
	"io"
	"net/http"
)

type ClientMock struct {
	body                []byte
	err                 error
	validateRequestFunc func(req *http.Request) error
}

func (c ClientMock) Do(req *http.Request) (*http.Response, error) {
	err := c.validateRequestFunc(req)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		Body: io.NopCloser(bytes.NewReader(c.body)),
	}, c.err
}
