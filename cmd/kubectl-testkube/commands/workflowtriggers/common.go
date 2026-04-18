// Package workflowtriggers provides CLI commands for WorkflowTrigger (v2) resources.
// Mirrors the webhooks/ and testtriggers/ layout: one file per CRUD operation, plus
// this common helper file for flag wiring and input decoding.
package workflowtriggers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	workflowtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/workflowtriggers"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

// readWorkflowTriggerFromInput reads the request body from stdin or a file and
// decodes it as JSON/YAML. Both flat API shape and CRD shape (with spec:) are
// accepted — we detect and map.
func readWorkflowTriggerFromInput(file string) (testkube.WorkflowTrigger, error) {
	var src io.Reader
	if file == "" || file == "-" {
		src = os.Stdin
	} else {
		f, err := os.Open(file)
		if err != nil {
			return testkube.WorkflowTrigger{}, fmt.Errorf("opening %s: %w", file, err)
		}
		defer f.Close()
		src = f
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return testkube.WorkflowTrigger{}, fmt.Errorf("reading input: %w", err)
	}

	// Try CRD shape first (kubectl apply YAML). If the top-level has `spec:`
	// it's the CRD form; map to the flat API shape.
	var probe map[string]interface{}
	if err := yaml.Unmarshal(data, &probe); err == nil {
		if _, hasSpec := probe["spec"]; hasSpec {
			var crd workflowtriggersv1.WorkflowTrigger
			if err := yaml.Unmarshal(data, &crd); err != nil {
				return testkube.WorkflowTrigger{}, fmt.Errorf("parsing CRD: %w", err)
			}
			return workflowtriggersmapper.MapCRDToAPI(&crd), nil
		}
	}

	// Flat shape — try JSON, then YAML.
	var trigger testkube.WorkflowTrigger
	if err := json.Unmarshal(data, &trigger); err == nil {
		return trigger, nil
	}
	if err := yaml.Unmarshal(data, &trigger); err != nil {
		return testkube.WorkflowTrigger{}, fmt.Errorf("parsing input: %w", err)
	}
	return trigger, nil
}
