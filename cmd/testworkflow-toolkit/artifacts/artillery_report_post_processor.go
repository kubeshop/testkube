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

const maxArtilleryReportProbeBytes int64 = 16 << 20

// ArtilleryReportPostProcessor checks JSON files for Artillery reports and sends them to the cloud.
type ArtilleryReportPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func NewArtilleryReportPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *ArtilleryReportPostProcessor {
	return &ArtilleryReportPostProcessor{
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

func (p *ArtilleryReportPostProcessor) Start() error {
	return nil
}

func (p *ArtilleryReportPostProcessor) Add(path string) error {
	if err := p.add(path); err != nil {
		fmt.Printf("warn: artillery report processing: %s: %s\n", path, err)
	}
	return nil
}

func (p *ArtilleryReportPostProcessor) End() error {
	return nil
}

func (p *ArtilleryReportPostProcessor) add(path string) error {
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
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get file info for %s", path)
	}
	if !isJSONFile(stat) {
		return nil
	}

	if !hasArtilleryReportShape(file, maxArtilleryReportProbeBytes) {
		return nil
	}
	if err := file.Close(); err != nil {
		return errors.Wrapf(err, "failed to close %s", path)
	}
	file = nil
	reportFile, err := p.fs.OpenFileRO(absPath)
	if err != nil {
		return errors.Wrapf(err, "failed to reopen %s", path)
	}
	defer func() { _ = reportFile.Close() }()

	data, err := io.ReadAll(reportFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", path)
	}
	if !isArtilleryReport(data) {
		return nil
	}

	fmt.Printf("Processing Artillery report: %s\n", ui.LightCyan(path))
	if err := p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, uploadPath, data); err != nil {
		return errors.Wrapf(err, "failed to send Artillery report %s", stat.Name())
	}
	return nil
}

func hasArtilleryReportShape(reader io.Reader, maxBytes int64) bool {
	limited := &io.LimitedReader{R: reader, N: maxBytes}
	decoder := json.NewDecoder(limited)

	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return false
	}

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return false
		}
		key, ok := keyToken.(string)
		if !ok {
			return false
		}
		if key == "aggregate" {
			return artilleryAggregateHasMetricShape(decoder)
		}
		if err := skipJSONValue(decoder); err != nil {
			return false
		}
	}
	return false
}

func artilleryAggregateHasMetricShape(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return false
	}

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return false
		}
		key, ok := keyToken.(string)
		if !ok {
			return false
		}
		if key == "counters" || key == "rates" || key == "summaries" || key == "histograms" {
			if jsonValueHasNumber(decoder) {
				return true
			}
			continue
		}
		if err := skipJSONValue(decoder); err != nil {
			return false
		}
	}
	if _, err := decoder.Token(); err != nil {
		return false
	}
	return false
}

func isArtilleryReport(data []byte) bool {
	var report struct {
		Aggregate artilleryAggregate `json:"aggregate"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return false
	}
	return artilleryAggregateHasMetrics(report.Aggregate)
}

type artilleryAggregate struct {
	Counters   map[string]float64            `json:"counters"`
	Rates      map[string]float64            `json:"rates"`
	Summaries  map[string]map[string]float64 `json:"summaries"`
	Histograms map[string]map[string]float64 `json:"histograms"`
}

func artilleryAggregateHasMetrics(aggregate artilleryAggregate) bool {
	return len(aggregate.Counters) > 0 ||
		len(aggregate.Rates) > 0 ||
		len(aggregate.Summaries) > 0 ||
		len(aggregate.Histograms) > 0
}
