package marketplace

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const objectConfigYAML = `apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: example
spec:
  config:
    host:
      type: string
      default: "localhost"
      description: "Host to connect to"
    port:
      type: string
      default: "5432"
    password:
      type: string
      default: ""
      sensitive: true
      description: "Password"
  steps:
  - name: run
    shell: echo hi
`

const scalarConfigYAML = `spec:
  config:
    host: "localhost"
    port: "5432"
`

func TestExtractParameters_ObjectShape(t *testing.T) {
	params, err := ExtractParameters([]byte(objectConfigYAML))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d: %+v", len(params), params)
	}
	want := map[string]Parameter{
		"host":     {Key: "host", Default: "localhost", Value: "localhost", Type: "string", Description: "Host to connect to"},
		"port":     {Key: "port", Default: "5432", Value: "5432", Type: "string"},
		"password": {Key: "password", Default: "", Value: "", Type: "string", Description: "Password", Sensitive: true},
	}
	for _, p := range params {
		w, ok := want[p.Key]
		if !ok {
			t.Errorf("unexpected param: %+v", p)
			continue
		}
		if p != w {
			t.Errorf("param %s mismatch:\n got: %+v\nwant: %+v", p.Key, p, w)
		}
	}
}

func TestExtractParameters_ScalarShape(t *testing.T) {
	params, err := ExtractParameters([]byte(scalarConfigYAML))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Key != "host" || params[0].Default != "localhost" || params[0].Value != "localhost" {
		t.Errorf("unexpected host param: %+v", params[0])
	}
}

func TestExtractParameters_NoConfig(t *testing.T) {
	params, err := ExtractParameters([]byte("spec:\n  steps: []\n"))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}
	if len(params) != 0 {
		t.Errorf("expected no params, got %+v", params)
	}
}

func TestApplyParameters_RewritesDefaults(t *testing.T) {
	params, err := ExtractParameters([]byte(objectConfigYAML))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}
	for i := range params {
		switch params[i].Key {
		case "host":
			params[i].Value = "my-host.svc"
		case "password":
			params[i].Value = "s3cret"
		}
	}

	out, err := ApplyParameters([]byte(objectConfigYAML), params)
	if err != nil {
		t.Fatalf("ApplyParameters: %v", err)
	}

	cfg := decodeSpecConfig(t, out)
	host := cfg["host"].(map[string]any)
	if host["default"] != "my-host.svc" {
		t.Errorf("host default not updated: %v", host["default"])
	}
	port := cfg["port"].(map[string]any)
	if port["default"] != "5432" {
		t.Errorf("port default should be unchanged: %v", port["default"])
	}
	pw := cfg["password"].(map[string]any)
	if pw["default"] != "s3cret" {
		t.Errorf("password default not updated: %v", pw["default"])
	}
	if pw["sensitive"] != true {
		t.Errorf("password sensitive should remain true: %v", pw["sensitive"])
	}

	if !strings.Contains(string(out), "echo hi") {
		t.Error("expected steps to be preserved in re-encoded YAML")
	}
}

func TestApplyParameters_DropsSensitiveWhenEmpty(t *testing.T) {
	params, err := ExtractParameters([]byte(objectConfigYAML))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}

	out, err := ApplyParameters([]byte(objectConfigYAML), params)
	if err != nil {
		t.Fatalf("ApplyParameters: %v", err)
	}
	cfg := decodeSpecConfig(t, out)
	pw := cfg["password"].(map[string]any)
	if _, exists := pw["sensitive"]; exists {
		t.Errorf("expected sensitive to be dropped when value is empty, got %+v", pw)
	}
}

func TestApplyParameters_ScalarEntries(t *testing.T) {
	params, err := ExtractParameters([]byte(scalarConfigYAML))
	if err != nil {
		t.Fatalf("ExtractParameters: %v", err)
	}
	for i := range params {
		if params[i].Key == "host" {
			params[i].Value = "new-host"
		}
	}

	out, err := ApplyParameters([]byte(scalarConfigYAML), params)
	if err != nil {
		t.Fatalf("ApplyParameters: %v", err)
	}
	cfg := decodeSpecConfig(t, out)
	if cfg["host"] != "new-host" {
		t.Errorf("expected host scalar to be new-host, got %v", cfg["host"])
	}
	if cfg["port"] != "5432" {
		t.Errorf("expected port scalar preserved, got %v", cfg["port"])
	}
}

func TestParseSetFlags(t *testing.T) {
	params := []Parameter{
		{Key: "host", Default: "a", Value: "a"},
		{Key: "port", Default: "1", Value: "1"},
	}
	out, err := ParseSetFlags(params, []string{"host=b", "port=2"})
	if err != nil {
		t.Fatalf("ParseSetFlags: %v", err)
	}
	if out[0].Value != "b" || out[1].Value != "2" {
		t.Errorf("unexpected values: %+v", out)
	}
	if params[0].Value != "a" || params[1].Value != "1" {
		t.Errorf("ParseSetFlags should not mutate input")
	}
}

func TestParseSetFlags_Errors(t *testing.T) {
	params := []Parameter{{Key: "host", Default: "a", Value: "a"}}
	if _, err := ParseSetFlags(params, []string{"no-equals"}); err == nil {
		t.Error("expected error for missing equals")
	}
	if _, err := ParseSetFlags(params, []string{"unknown=x"}); err == nil {
		t.Error("expected error for unknown key")
	}
	out, err := ParseSetFlags(params, []string{"host=with=equals"})
	if err != nil {
		t.Fatalf("ParseSetFlags with embedded equals: %v", err)
	}
	if out[0].Value != "with=equals" {
		t.Errorf("expected value with embedded equals preserved, got %q", out[0].Value)
	}
}

func decodeSpecConfig(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var doc struct {
		Spec struct {
			Config map[string]any `yaml:"config"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decoding output yaml: %v\n%s", err, data)
	}
	return doc.Spec.Config
}
