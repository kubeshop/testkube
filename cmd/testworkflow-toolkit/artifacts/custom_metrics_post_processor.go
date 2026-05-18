package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

const customMetricsReportKind = "testkube.custom-metrics"

// CustomMetricsPostProcessor checks JSON files for granular insight reports and sends them to the cloud.
type CustomMetricsPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func NewCustomMetricsPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *CustomMetricsPostProcessor {
	return &CustomMetricsPostProcessor{
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

func (p *CustomMetricsPostProcessor) Start() error {
	return nil
}

func (p *CustomMetricsPostProcessor) Add(path string) error {
	if err := p.add(path); err != nil {
		fmt.Printf("warn: custom metrics processing: %s: %s\n", path, err)
	}
	return nil
}

func (p *CustomMetricsPostProcessor) End() error {
	return nil
}

func (p *CustomMetricsPostProcessor) add(path string) error {
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
	if !isGranularMetricsReport(data) {
		return nil
	}

	fmt.Printf("Processing metrics report: %s\n", ui.LightCyan(path))
	if err := p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, uploadPath, data); err != nil {
		return errors.Wrapf(err, "failed to send metrics report %s", stat.Name())
	}
	return nil
}

func isGranularMetricsReport(data []byte) bool {
	return isCustomMetricsReport(data) ||
		isArtilleryReport(data) ||
		isPlaywrightJSONReport(data) ||
		isCypressJSONReport(data)
}

func isCustomMetricsReport(data []byte) bool {
	var report struct {
		Kind    string `json:"kind"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return false
	}
	return report.Kind == customMetricsReportKind && report.Version == "v1"
}

func isArtilleryReport(data []byte) bool {
	var report struct {
		Aggregate struct {
			Counters   map[string]float64            `json:"counters"`
			Rates      map[string]float64            `json:"rates"`
			Summaries  map[string]map[string]float64 `json:"summaries"`
			Histograms map[string]map[string]float64 `json:"histograms"`
		} `json:"aggregate"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return false
	}
	return len(report.Aggregate.Counters) > 0 ||
		len(report.Aggregate.Rates) > 0 ||
		len(report.Aggregate.Summaries) > 0 ||
		len(report.Aggregate.Histograms) > 0
}

func isPlaywrightJSONReport(data []byte) bool {
	var report struct {
		Suites []struct {
			Specs []struct {
				Tests []struct{} `json:"tests"`
			} `json:"specs"`
			Suites []struct {
				Specs []struct {
					Tests []struct{} `json:"tests"`
				} `json:"specs"`
			} `json:"suites"`
		} `json:"suites"`
	}
	if err := json.Unmarshal(data, &report); err != nil || len(report.Suites) == 0 {
		return false
	}
	for _, suite := range report.Suites {
		for _, spec := range suite.Specs {
			if len(spec.Tests) > 0 {
				return true
			}
		}
		for _, child := range suite.Suites {
			for _, spec := range child.Specs {
				if len(spec.Tests) > 0 {
					return true
				}
			}
		}
	}
	return false
}

func isCypressJSONReport(data []byte) bool {
	var report struct {
		Stats   *struct{}  `json:"stats"`
		Tests   []struct{} `json:"tests"`
		Results []struct{} `json:"results"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return false
	}
	return report.Stats != nil && (len(report.Tests) > 0 || len(report.Results) > 0)
}
