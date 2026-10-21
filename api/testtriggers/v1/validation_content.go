package v1

import (
	"fmt"
	"net/url"
	"slices"
	"time"
)

const (
	// DefaultHTTPContentInterval is the poll interval of an HTTP content trigger that sets none.
	DefaultHTTPContentInterval = time.Minute
	// MinHTTPContentInterval is the shortest poll interval an HTTP content trigger accepts.
	MinHTTPContentInterval = 10 * time.Second
	// DefaultHTTPContentTimeout is the request timeout of an HTTP content trigger that sets none.
	DefaultHTTPContentTimeout = 10 * time.Second
)

// HTTPContentMethods are the request methods an HTTP content trigger accepts.
var HTTPContentMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// validateContentSelector checks that a content trigger watches exactly one source and only
// fires the events of that source.
func validateContentSelector(selector *TestTriggerContentSelector, events []TestTriggerEvent) []error {
	hasGit := selector != nil && selector.Git != nil
	hasHTTP := selector != nil && selector.HTTP != nil

	switch {
	case hasGit && hasHTTP:
		return []error{fmt.Errorf("resource %q accepts only one of contentSelector.git or contentSelector.http", TestTriggerResourceContent)}
	case hasHTTP:
		var errs []error
		for _, event := range events {
			if event != TestTriggerEventHTTPChanged {
				errs = append(errs, fmt.Errorf("resource %q with contentSelector.http requires event to be %q", TestTriggerResourceContent, TestTriggerEventHTTPChanged))
				break
			}
		}
		return append(errs, selector.HTTP.Validate()...)
	default:
		var errs []error
		for _, event := range events {
			if event != TestTriggerEventGitPush && event != TestTriggerEventGitTagPush && event != TestTriggerEventGitPullRequest {
				errs = append(errs, fmt.Errorf("resource %q requires event to be one of %q, %q, or %q", TestTriggerResourceContent, TestTriggerEventGitPush, TestTriggerEventGitTagPush, TestTriggerEventGitPullRequest))
				break
			}
		}
		if !hasGit || selector.Git.Uri == "" {
			errs = append(errs, fmt.Errorf("resource %q requires contentSelector.git.uri or contentSelector.http.url", TestTriggerResourceContent))
		}
		return errs
	}
}

// Validate checks an HTTP content source.
func (h *TestTriggerContentHTTPSpec) Validate() []error {
	var errs []error

	if h.URL == "" {
		errs = append(errs, fmt.Errorf("contentSelector.http.url is required"))
	} else if u, err := url.Parse(h.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, fmt.Errorf("contentSelector.http.url %q must be an absolute http or https URL", h.URL))
	}
	if h.Method != "" && !slices.Contains(HTTPContentMethods, h.Method) {
		errs = append(errs, fmt.Errorf("contentSelector.http.method %q must be one of %v", h.Method, HTTPContentMethods))
	}
	if h.Interval != nil && h.Interval.Duration < MinHTTPContentInterval {
		errs = append(errs, fmt.Errorf("contentSelector.http.interval must be at least %s", MinHTTPContentInterval))
	}
	if h.Timeout != nil && h.Timeout.Duration <= 0 {
		errs = append(errs, fmt.Errorf("contentSelector.http.timeout must be positive"))
	}
	for name := range h.Headers {
		if name == "" {
			errs = append(errs, fmt.Errorf("contentSelector.http.headers cannot have an empty name"))
			break
		}
	}
	for i, header := range h.HeadersFrom {
		if header.Name == "" {
			errs = append(errs, fmt.Errorf("contentSelector.http.headersFrom[%d].name is required", i))
		}
		if header.ValueFrom.SecretKeyRef == nil && header.ValueFrom.ConfigMapKeyRef == nil {
			errs = append(errs, fmt.Errorf("contentSelector.http.headersFrom[%d].valueFrom requires secretKeyRef or configMapKeyRef", i))
		}
	}
	for _, status := range h.ExpectedStatus {
		if status < 100 || status > 599 {
			errs = append(errs, fmt.Errorf("contentSelector.http.expectedStatus %d is not an HTTP status code", status))
		}
	}
	if h.Select != nil {
		if h.Select.JSONPath != "" && h.Select.Header != "" {
			errs = append(errs, fmt.Errorf("contentSelector.http.select accepts only one of jsonPath or header"))
		}
		if h.Select.JSONPath != "" && (!MatchPathPattern.MatchString(h.Select.JSONPath) || MatchPathBracketPattern.MatchString(h.Select.JSONPath)) {
			errs = append(errs, fmt.Errorf("contentSelector.http.select.jsonPath %q is not a valid dot-path (e.g. .data.version)", h.Select.JSONPath))
		}
	}
	return errs
}
