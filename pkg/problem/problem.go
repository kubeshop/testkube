// Package problem implements the "Problem Details for HTTP APIs" response
// model defined by RFC 9457 (previously RFC 7807), which the Testkube API
// uses to report errors.
package problem

import "net/http"

// Problem describes an HTTP API error response as defined by RFC 9457.
type Problem struct {
	// Type is a URI reference that identifies the problem type.
	Type string `json:"type"`
	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`
	// Status is the HTTP status code for this occurrence of the problem.
	Status int `json:"status,omitempty"`
	// Detail is a human-readable explanation specific to this occurrence
	// of the problem.
	Detail string `json:"detail,omitempty"`
	// Instance is a URI reference that identifies the specific occurrence
	// of the problem.
	Instance string `json:"instance,omitempty"`
}

// New returns a Problem for the given HTTP status code and detail message,
// with the standard "about:blank" type and the status text as the title.
func New(status int, details string) Problem {
	return Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: details,
	}
}
