package marketplace

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kubeshop/testkube/pkg/marketplace"
)

// stubPrompter returns canned answers keyed by parameter Key and records
// which params were asked, in order. An answer of "" simulates the user
// hitting Enter with no input.
type stubPrompter struct {
	answers      map[string]string
	asked        []marketplace.Parameter
	err          map[string]error
	confirmAns   bool
	confirmErr   error
	confirmCalls int
	confirmMsg   string
}

func (s *stubPrompter) Prompt(p marketplace.Parameter) (string, error) {
	s.asked = append(s.asked, p)
	if s.err != nil {
		if e, ok := s.err[p.Key]; ok {
			return "", e
		}
	}
	return s.answers[p.Key], nil
}

func (s *stubPrompter) Confirm(message string, _ bool) (bool, error) {
	s.confirmCalls++
	s.confirmMsg = message
	if s.confirmErr != nil {
		return false, s.confirmErr
	}
	return s.confirmAns, nil
}

func TestPromptForParameters(t *testing.T) {
	type want struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		params  []marketplace.Parameter
		answers map[string]string
		want    []want
	}{
		{
			name: "user-provided values override defaults",
			params: []marketplace.Parameter{
				{Key: "host", Default: "localhost", Value: "localhost", Type: "string"},
				{Key: "port", Default: "6379", Value: "6379", Type: "integer"},
			},
			answers: map[string]string{
				"host": "redis.prod.svc",
				"port": "6380",
			},
			want: []want{
				{"host", "redis.prod.svc"},
				{"port", "6380"},
			},
		},
		{
			name: "empty input preserves current value",
			params: []marketplace.Parameter{
				{Key: "host", Default: "localhost", Value: "localhost"},
				{Key: "port", Default: "6379", Value: "6379"},
			},
			answers: map[string]string{
				"host": "",
				"port": "",
			},
			want: []want{
				{"host", "localhost"},
				{"port", "6379"},
			},
		},
		{
			name: "empty input keeps prior --set value",
			params: []marketplace.Parameter{
				{Key: "host", Default: "localhost", Value: "redis.stage.svc"},
				{Key: "port", Default: "6379", Value: "6379"},
			},
			answers: map[string]string{
				"host": "",
				"port": "6400",
			},
			want: []want{
				{"host", "redis.stage.svc"},
				{"port", "6400"},
			},
		},
		{
			name: "sensitive params are forwarded to the prompter",
			params: []marketplace.Parameter{
				{Key: "password", Type: "string", Sensitive: true},
			},
			answers: map[string]string{
				"password": "hunter2",
			},
			want: []want{
				{"password", "hunter2"},
			},
		},
		{
			name:    "empty parameter list is a no-op",
			params:  nil,
			answers: map[string]string{},
			want:    nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubPrompter{answers: tc.answers}
			var buf bytes.Buffer
			got, err := promptForParameters(&buf, tc.params, stub)
			if err != nil {
				t.Fatalf("promptForParameters: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %d want %d", len(got), len(tc.want))
			}
			for i, w := range tc.want {
				if got[i].Key != w.key || got[i].Value != w.value {
					t.Errorf("param %d: got (%q, %q) want (%q, %q)", i, got[i].Key, got[i].Value, w.key, w.value)
				}
			}
			if len(tc.params) > 0 && !strings.Contains(buf.String(), "workflow exposes") {
				t.Errorf("expected prompt header in output, got %q", buf.String())
			}
		})
	}
}

func TestPromptForParameters_PrompterErrorPropagates(t *testing.T) {
	stub := &stubPrompter{
		answers: map[string]string{},
		err:     map[string]error{"db-password": errors.New("ctrl-c")},
	}
	params := []marketplace.Parameter{
		{Key: "db-password", Sensitive: true},
	}
	_, err := promptForParameters(&bytes.Buffer{}, params, stub)
	if err == nil {
		t.Fatal("expected error to propagate from prompter")
	}
	if !strings.Contains(err.Error(), "db-password") {
		t.Errorf("error should mention the failing parameter, got %v", err)
	}
}

func TestPromptForParameters_RendersHeaderAndDescription(t *testing.T) {
	stub := &stubPrompter{answers: map[string]string{"host": ""}}
	var buf bytes.Buffer
	_, err := promptForParameters(&buf, []marketplace.Parameter{
		{Key: "host", Type: "string", Description: "Database host FQDN", Value: "db.svc"},
	}, stub)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "1/1  host (string)") {
		t.Errorf("missing header in output: %q", out)
	}
	if !strings.Contains(out, "Database host FQDN") {
		t.Errorf("missing description in output: %q", out)
	}
	if strings.Contains(out, "[sensitive]") {
		t.Errorf("unexpected [sensitive] marker for non-sensitive param: %q", out)
	}
}

func TestPromptForParameters_MarksSensitiveParams(t *testing.T) {
	stub := &stubPrompter{answers: map[string]string{"token": ""}}
	var buf bytes.Buffer
	_, err := promptForParameters(&buf, []marketplace.Parameter{
		{Key: "token", Sensitive: true},
	}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[sensitive]") {
		t.Errorf("expected [sensitive] marker for sensitive param, got %q", buf.String())
	}
}

func TestPromptForParameters_DefaultsTypeToString(t *testing.T) {
	stub := &stubPrompter{answers: map[string]string{"untyped": ""}}
	var buf bytes.Buffer
	_, err := promptForParameters(&buf, []marketplace.Parameter{
		{Key: "untyped"},
	}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "untyped (string)") {
		t.Errorf("expected fallback type 'string' in header, got %q", buf.String())
	}
}
