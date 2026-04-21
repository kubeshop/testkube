// Package testtriggers provides CLI commands for TestTrigger resources.
// Mirrors the workflowtriggers/ layout: one file per CRUD operation, plus this
// common helper file for flag wiring and input decoding.
package testtriggers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"

	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

// readTestTriggerFromInput reads the request body from stdin or a file and
// decodes it as JSON/YAML. Both flat API shape (TestTriggerUpsertRequest) and
// the CRD shape (with spec:) are accepted — we detect and map.
func readTestTriggerFromInput(file string) (testkube.TestTriggerUpsertRequest, error) {
	var src io.Reader
	if file == "" || file == "-" {
		src = os.Stdin
	} else {
		f, err := os.Open(file)
		if err != nil {
			return testkube.TestTriggerUpsertRequest{}, fmt.Errorf("opening %s: %w", file, err)
		}
		defer f.Close()
		src = f
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return testkube.TestTriggerUpsertRequest{}, fmt.Errorf("reading input: %w", err)
	}

	// Try CRD shape first (kubectl apply YAML). If the top-level has `spec:`
	// it's the CRD form; map to the flat upsert request.
	var probe map[string]interface{}
	if err := yaml.Unmarshal(data, &probe); err == nil {
		if _, hasSpec := probe["spec"]; hasSpec {
			var crd testtriggersv1.TestTrigger
			if err := yaml.Unmarshal(data, &crd); err != nil {
				return testkube.TestTriggerUpsertRequest{}, fmt.Errorf("parsing CRD: %w", err)
			}
			return testtriggersmapper.MapTestTriggerCRDToTestTriggerUpsertRequest(crd), nil
		}
	}

	// Flat shape — try JSON, then YAML.
	var req testkube.TestTriggerUpsertRequest
	if err := json.Unmarshal(data, &req); err == nil {
		return req, nil
	}
	if err := yaml.Unmarshal(data, &req); err != nil {
		return testkube.TestTriggerUpsertRequest{}, fmt.Errorf("parsing input: %w", err)
	}
	return req, nil
}

// toCreateOptions converts an upsert request to the client's create options.
// Both types are aliases over TestTriggerUpsertRequest.
func toCreateOptions(req testkube.TestTriggerUpsertRequest) apiv1.CreateTestTriggerOptions {
	return apiv1.CreateTestTriggerOptions(req)
}

// toUpdateOptions converts an upsert request to the client's update options.
func toUpdateOptions(req testkube.TestTriggerUpsertRequest) apiv1.UpdateTestTriggerOptions {
	return apiv1.UpdateTestTriggerOptions(req)
}
