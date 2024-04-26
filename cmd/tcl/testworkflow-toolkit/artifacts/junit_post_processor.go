package artifacts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/tcl/cloudtcl/data/testworkflow"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

// JUnitPostProcessor is a post-processor that checks XML files for JUnit reports and sends them to the cloud.
type JUnitPostProcessor struct {
	fs     filesystem.FileSystem
	client cloudexecutor.Executor
}

func NewJUnitPostProcessor(fs filesystem.FileSystem, client cloudexecutor.Executor) *JUnitPostProcessor {
	return &JUnitPostProcessor{fs: fs, client: client}
}

func (p *JUnitPostProcessor) Start() error {
	return nil
}

// Add checks if the file is a JUnit report and sends it to the cloud.
func (p *JUnitPostProcessor) Add(path string) error {
	file, err := p.fs.OpenFileRO(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", path)
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to get file info for %s", path)
	}
	if ok := isXMLFile(stat); !ok {
		return nil
	}
	xmlData, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", path)
	}
	ok, err := isJUnitReport(xmlData)
	if err != nil {
		return errors.Wrapf(err, "failed to check if %s is a JUnit report", path)
	}
	if !ok {
		return nil
	}
	fmt.Printf("Processing JUnit report: %s\n", ui.LightCyan(path))
	if err := p.sendJUnitReport(path, xmlData); err != nil {
		return errors.Wrapf(err, "failed to send JUnit report %s", stat.Name())
	}
	return nil
}

// sendJUnitReport sends the JUnit report to the Agent gRPC API.
func (p *JUnitPostProcessor) sendJUnitReport(path string, report []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := p.client.Execute(ctx, testworkflow.CmdTestWorkflowExecutionAddReport, &testworkflow.ExecutionsAddReportRequest{
		ID:           env.ExecutionId(),
		WorkflowName: env.WorkflowName(),
		WorkflowStep: env.Ref(),
		Filepath:     path,
		Report:       report,
	})
	return err
}

// isXMLFile checks if the file is an XML file based on the extension.
func isXMLFile(stat fs.FileInfo) bool {
	if stat.IsDir() || stat.Size() == 0 {
		return false
	}

	return strings.HasSuffix(stat.Name(), ".xml")
}

// isJUnitReport checks if the file starts with a JUnit XML tag
func isJUnitReport(xmlData []byte) (bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(xmlData))
	// Read only the first few lines of the file
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line) // Remove leading and trailing whitespace

		// Skip comments and declarations
		if strings.HasPrefix(line, "<!--") || strings.HasPrefix(line, "<?xml") {
			continue
		}
		if strings.Contains(line, "<testsuite") || strings.Contains(line, "<testsuites") {
			return true, nil
		}
		if strings.Contains(line, "<") { // Stop if any non-JUnit tag is found
			break
		}
	}

	return false, scanner.Err()
}

func (p *JUnitPostProcessor) End() error {
	return nil
}
