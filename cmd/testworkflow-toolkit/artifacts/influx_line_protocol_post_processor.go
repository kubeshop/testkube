package artifacts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utilization/core"
)

const maxInfluxLineProtocolProbeBytes int64 = 16 << 20

var influxLineProtocolParser = core.NewInfluxDBLineProtocolParser()

// InfluxLineProtocolPostProcessor checks files for InfluxDB line protocol records and sends them to the cloud.
type InfluxLineProtocolPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func NewInfluxLineProtocolPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *InfluxLineProtocolPostProcessor {
	return &InfluxLineProtocolPostProcessor{
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

func (p *InfluxLineProtocolPostProcessor) Start() error {
	return nil
}

func (p *InfluxLineProtocolPostProcessor) Add(path string) error {
	if err := p.add(path); err != nil {
		fmt.Printf("warn: influx line protocol processing: %s: %s\n", path, err)
	}
	return nil
}

func (p *InfluxLineProtocolPostProcessor) End() error {
	return nil
}

func (p *InfluxLineProtocolPostProcessor) add(path string) error {
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
	if !isInfluxLineProtocolFile(stat) {
		return nil
	}

	if !hasInfluxLineProtocolShape(file, maxInfluxLineProtocolProbeBytes) {
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
	if !isInfluxLineProtocolReport(data) {
		return nil
	}

	fmt.Printf("Processing InfluxDB line protocol report: %s\n", ui.LightCyan(path))
	if err := p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, uploadPath, data); err != nil {
		return errors.Wrapf(err, "failed to send InfluxDB line protocol report %s", stat.Name())
	}
	return nil
}

func isInfluxLineProtocolFile(stat fs.FileInfo) bool {
	if stat.IsDir() || stat.Size() == 0 {
		return false
	}
	name := strings.ToLower(stat.Name())
	return strings.HasSuffix(name, ".influx") || strings.HasSuffix(name, ".lp")
}

func hasInfluxLineProtocolShape(reader io.Reader, maxBytes int64) bool {
	limited := &io.LimitedReader{R: reader, N: maxBytes}
	return influxLineProtocolHasValidLine(limited)
}

func isInfluxLineProtocolReport(data []byte) bool {
	return influxLineProtocolHasValidLine(bytes.NewReader(data))
}

func influxLineProtocolHasValidLine(reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, err := influxLineProtocolParser.Parse([]byte(line)); err == nil {
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("warn: influx line protocol scanning: %s\n", err)
	}
	return false
}
