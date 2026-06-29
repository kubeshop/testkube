package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

const maxJMeterStatisticsProbeBytes int64 = 16 << 20

// JMeterStatisticsPostProcessor checks JMeter dashboard statistics files and sends them to the cloud.
type JMeterStatisticsPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func NewJMeterStatisticsPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *JMeterStatisticsPostProcessor {
	return &JMeterStatisticsPostProcessor{
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

func (p *JMeterStatisticsPostProcessor) Start() error {
	return nil
}

func (p *JMeterStatisticsPostProcessor) Add(path string) error {
	if err := p.add(path); err != nil {
		fmt.Printf("warn: jmeter statistics processing: %s: %s\n", path, err)
	}
	return nil
}

func (p *JMeterStatisticsPostProcessor) End() error {
	return nil
}

func (p *JMeterStatisticsPostProcessor) add(path string) error {
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
	if !isJSONFile(stat) || !isJMeterStatisticsFilename(stat.Name()) {
		return nil
	}

	if !hasJMeterStatisticsShape(file, maxJMeterStatisticsProbeBytes) {
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
	if !isJMeterStatisticsReport(data) {
		return nil
	}

	fmt.Printf("Processing JMeter statistics report: %s\n", ui.LightCyan(path))
	if err := p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, uploadPath, data); err != nil {
		return errors.Wrapf(err, "failed to send JMeter statistics report %s", stat.Name())
	}
	return nil
}

func isJMeterStatisticsFilename(name string) bool {
	return strings.EqualFold(name, "statistics.json")
}

func hasJMeterStatisticsShape(reader io.Reader, maxBytes int64) bool {
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
		if key == "Total" {
			return jmeterStatisticsEntryHasMetricShape(decoder)
		}
		if err := skipJSONValue(decoder); err != nil {
			return false
		}
	}
	return false
}

func jmeterStatisticsEntryHasMetricShape(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return false
	}

	hasTransaction := false
	hasSampleCount := false
	hasMetric := false
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return false
		}
		key, ok := keyToken.(string)
		if !ok {
			return false
		}
		switch key {
		case "transaction":
			var transaction string
			if err := decoder.Decode(&transaction); err != nil {
				return false
			}
			hasTransaction = transaction == "Total"
		case "sampleCount":
			hasSampleCount = jsonValueHasNumber(decoder)
		case "errorCount", "errorPct", "meanResTime", "medianResTime", "minResTime", "maxResTime", "pct1ResTime", "pct2ResTime", "pct3ResTime", "throughput", "receivedKBytesPerSec", "sentKBytesPerSec":
			hasMetric = jsonValueHasNumber(decoder)
		default:
			if err := skipJSONValue(decoder); err != nil {
				return false
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return false
	}
	return hasTransaction && hasSampleCount && hasMetric
}

func isJMeterStatisticsReport(data []byte) bool {
	var report map[string]jmeterStatisticsEntry
	if err := json.Unmarshal(data, &report); err != nil {
		return false
	}
	total, ok := report["Total"]
	return ok && total.Transaction == "Total" && total.hasStatisticsMetrics()
}

type jmeterStatisticsEntry struct {
	Transaction          string   `json:"transaction"`
	SampleCount          *float64 `json:"sampleCount"`
	ErrorCount           *float64 `json:"errorCount"`
	ErrorPct             *float64 `json:"errorPct"`
	MeanResTime          *float64 `json:"meanResTime"`
	MedianResTime        *float64 `json:"medianResTime"`
	MinResTime           *float64 `json:"minResTime"`
	MaxResTime           *float64 `json:"maxResTime"`
	Pct1ResTime          *float64 `json:"pct1ResTime"`
	Pct2ResTime          *float64 `json:"pct2ResTime"`
	Pct3ResTime          *float64 `json:"pct3ResTime"`
	Throughput           *float64 `json:"throughput"`
	ReceivedKBytesPerSec *float64 `json:"receivedKBytesPerSec"`
	SentKBytesPerSec     *float64 `json:"sentKBytesPerSec"`
}

func (e jmeterStatisticsEntry) hasStatisticsMetrics() bool {
	return e.SampleCount != nil && (e.ErrorCount != nil ||
		e.ErrorPct != nil ||
		e.MeanResTime != nil ||
		e.MedianResTime != nil ||
		e.MinResTime != nil ||
		e.MaxResTime != nil ||
		e.Pct1ResTime != nil ||
		e.Pct2ResTime != nil ||
		e.Pct3ResTime != nil ||
		e.Throughput != nil ||
		e.ReceivedKBytesPerSec != nil ||
		e.SentKBytesPerSec != nil)
}
