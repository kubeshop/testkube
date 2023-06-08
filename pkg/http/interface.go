package http

import "net/http"

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
