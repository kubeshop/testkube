package testtriggers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

// readTestTriggerFromInput accepts both the flat UpsertRequest and the CRD shape (with spec:).
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

	var req testkube.TestTriggerUpsertRequest
	if err := json.Unmarshal(data, &req); err == nil {
		return req, nil
	}
	if err := yaml.Unmarshal(data, &req); err != nil {
		return testkube.TestTriggerUpsertRequest{}, fmt.Errorf("parsing input: %w", err)
	}
	return req, nil
}
