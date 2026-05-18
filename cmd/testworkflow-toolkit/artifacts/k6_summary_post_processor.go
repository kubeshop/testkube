package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

// K6SummaryPostProcessor checks JSON files for k6 summary reports and sends them to the cloud.
type K6SummaryPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func NewK6SummaryPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *K6SummaryPostProcessor {
	return &K6SummaryPostProcessor{
		fs:            fs,
		client:        client,
		environmentId: environmentId,
		executionId:   executionId,
		workflowName:  workflowName,
		stepRef:       stepRef,
		root:          root,
		pathPrefix:    pathPrefix,
	}
}

func (p *K6SummaryPostProcessor) Start() error {
	return nil
}

func (p *K6SummaryPostProcessor) Add(path string) error {
	if err := p.add(path); err != nil {
		fmt.Printf("warn: k6 summary processing: %s: %s\n", path, err)
	}
	return nil
}

func (p *K6SummaryPostProcessor) End() error {
	return nil
}

func (p *K6SummaryPostProcessor) add(path string) error {
	uploadPath := path
	if p.pathPrefix != "" {
		uploadPath = filepath.Join(p.pathPrefix, uploadPath)
	}
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(p.root, absPath)
	}
	file, err := p.fs.OpenFileRO(absPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", path)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get file info for %s", path)
	}
	if !isJSONFile(stat) {
		return nil
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", path)
	}
	if !isK6SummaryReport(data) {
		return nil
	}

	fmt.Printf("Processing k6 summary report: %s\n", ui.LightCyan(path))
	if err := p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, uploadPath, data); err != nil {
		return errors.Wrapf(err, "failed to send k6 summary report %s", stat.Name())
	}
	return nil
}

func isJSONFile(stat fs.FileInfo) bool {
	if stat.IsDir() || stat.Size() == 0 {
		return false
	}
	return strings.HasSuffix(stat.Name(), ".json")
}

func isK6SummaryReport(data []byte) bool {
	var report struct {
		Metrics map[string]map[string]json.RawMessage `json:"metrics"`
	}
	if err := json.Unmarshal(data, &report); err != nil || len(report.Metrics) == 0 {
		return false
	}
	for _, metric := range report.Metrics {
		if values, ok := metric["values"]; ok && hasNumericJSONValue(values) {
			return true
		}
		for key, value := range metric {
			if key == "type" || key == "contains" || key == "values" {
				continue
			}
			if hasNumericJSONValue(value) {
				return true
			}
		}
	}
	return false
}

func hasNumericJSONValue(data []byte) bool {
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(data, &nested); err == nil && len(nested) > 0 {
		for _, value := range nested {
			if hasNumericJSONValue(value) {
				return true
			}
		}
		return false
	}
	if strings.TrimSpace(string(data)) == "null" {
		return false
	}
	var value float64
	return json.Unmarshal(data, &value) == nil
}
