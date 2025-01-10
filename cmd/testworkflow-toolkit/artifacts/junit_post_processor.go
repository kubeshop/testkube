package artifacts

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"

	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

// JUnitPostProcessor is a post-processor that checks XML files for JUnit reports and sends them to the cloud.
type JUnitPostProcessor struct {
	fs         filesystem.FileSystem
	client     cloud.TestKubeCloudAPIClient
	apiKey     string
	root       string
	pathPrefix string
}

func NewJUnitPostProcessor(fs filesystem.FileSystem, client cloud.TestKubeCloudAPIClient, apiKey string, root, pathPrefix string) *JUnitPostProcessor {
	return &JUnitPostProcessor{fs: fs, client: client, apiKey: apiKey, root: root, pathPrefix: pathPrefix}
}

func (p *JUnitPostProcessor) Start() error {
	return nil
}

// Add checks if the file is a JUnit report and sends it to the cloud.
func (p *JUnitPostProcessor) Add(path string) error {
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
	if ok := isXMLFile(stat); !ok {
		return nil
	}

	// Read first 8KB
	const BYTE_SIZE_8KB = 8 * 1024
	buffer := make([]byte, BYTE_SIZE_8KB)
	n, err := io.ReadFull(file, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return errors.Wrapf(err, "failed to read initial content from %s", path)
	}
	buffer = buffer[:n] // Trim buffer to actual bytes read

	if !isJUnitReport(buffer) {
		return nil
	}

	// Read the rest of the file
	rest, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "failed to read remaining content from %s", path)
	}
	xmlData := append(buffer, rest...)

	fmt.Printf("Processing JUnit report: %s\n", ui.LightCyan(path))
	if err := p.sendJUnitReport(uploadPath, xmlData); err != nil {
		return errors.Wrapf(err, "failed to send JUnit report %s", stat.Name())
	}
	return nil
}

// sendJUnitReport sends the JUnit report to the Agent gRPC API.
func (p *JUnitPostProcessor) sendJUnitReport(path string, report []byte) error {
	if !env.IsNewExecutions() {
		return p.sendJUnitReportLegacy(path, report)
	}
	cfg := config.Config()
	md := metadata.New(map[string]string{"api-key": p.apiKey, "organization-id": cfg.Execution.OrganizationId, "agent-id": cfg.Worker.Connection.AgentID})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err := p.client.AppendExecutionReport(metadata.NewOutgoingContext(context.Background(), md), &cloud.AppendExecutionReportRequest{
		EnvironmentId: cfg.Execution.EnvironmentId,
		Id:            cfg.Execution.Id,
		Step:          config.Ref(), // TODO: think if it's valid for the parallel steps that have independent refs
		FilePath:      path,
		Report:        report,
	}, opts...)
	return err
}

func (p *JUnitPostProcessor) sendJUnitReportLegacy(path string, report []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := cloudexecutor.NewCloudGRPCExecutor(p.client, p.apiKey).Execute(ctx, testworkflow.CmdTestWorkflowExecutionAddReport, &testworkflow.ExecutionsAddReportRequest{
		ID:           config.ExecutionId(),
		WorkflowName: config.WorkflowName(),
		WorkflowStep: config.Ref(), // TODO: think if it's valid for the parallel steps that have independent refs
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

// isJUnitReport checks if the XML data is a JUnit report.
func isJUnitReport(xmlData []byte) bool {
	tags := []string{
		"<testsuite",
		"<testsuites",
	}

	content := string(xmlData)

	for _, tag := range tags {
		if strings.Contains(content, tag) {
			return true
		}
	}

	return false
}

func (p *JUnitPostProcessor) End() error {
	return nil
}
