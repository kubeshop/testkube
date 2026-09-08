package artifacts

import (
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
	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/ui"
)

// JUnitPostProcessor is a post-processor that checks XML files for JUnit reports and sends them to the cloud.
type JUnitPostProcessor struct {
	fs            filesystem.FileSystem
	client        controlplaneclient.ExecutionSelfClient
	root          string
	pathPrefix    string
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string

	// verdicts are what the containers that decided their steps' verdicts left
	// behind, keyed by the report file each covers. Loaded once in Start.
	verdicts map[string]testresults.ReportVerdict
}

func NewJUnitPostProcessor(
	fs filesystem.FileSystem,
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	root, pathPrefix string,
) *JUnitPostProcessor {
	return &JUnitPostProcessor{
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

// Start loads the verdicts recorded by the steps whose reports this will upload.
//
// A failure here is worth a warning and nothing more: without the verdicts the
// reports still upload, they just cannot say which failures were muted.
func (p *JUnitPostProcessor) Start() error {
	verdicts, err := testresults.ReadVerdicts()
	if err != nil {
		fmt.Printf("warn: JUnit processing: could not read the test case verdicts, muted counts will be missing: %s\n", err)
		return nil
	}
	p.verdicts = verdicts
	return nil
}

func (p *JUnitPostProcessor) Add(path string) error {
	err := p.add(path)
	if err != nil {
		fmt.Printf("warn: JUnit processing: %s: %s\n", path, err)
	}
	return nil
}

// Add checks if the file is a JUnit report and sends it to the cloud.
func (p *JUnitPostProcessor) add(path string) error {
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

	if !testresults.Sniff(buffer) {
		return nil
	}

	// Read the rest of the file
	rest, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "failed to read remaining content from %s", path)
	}
	xmlData := append(buffer, rest...)

	fmt.Printf("Processing JUnit report: %s\n", ui.LightCyan(path))

	// Parse here rather than leaving it to the control plane. The agent already
	// parses this report in the pod - muting decides the step's verdict and only
	// the pod can still change it - so re-parsing server-side is duplicated work
	// on a file that may be megabytes of XML.
	//
	// A report we cannot read is still worth uploading: the artifact is the
	// record, and a control plane that predates the parsed fields reads the raw
	// bytes anyway.
	var digest *testresults.Digest
	report, parseErr := testresults.Parse(bytes.NewReader(xmlData))
	if parseErr == nil {
		parsed := report.Digest()
		// The XML says which test cases failed and cannot say which of those
		// failures the workflow declared acceptable - that was decided in another
		// container, against this same file.
		if verdict, ok := p.verdictFor(absPath); ok {
			parsed = parsed.ApplyVerdict(verdict)
		}
		digest = &parsed
	} else {
		fmt.Printf("warn: JUnit report %s could not be parsed, sending it unparsed: %s\n", path, parseErr)
	}

	if err := p.sendJUnitReport(uploadPath, xmlData, digest); err != nil {
		return errors.Wrapf(err, "failed to send JUnit report %s", stat.Name())
	}
	return nil
}

// verdictFor finds the verdict recorded against a report file.
//
// The join is on the file itself rather than the step reference, because the
// artifacts stage carries a reference of its own, distinct from the stage that
// ran the tests. Both sides record absolute paths; a report nobody reached a
// verdict on - which is every report from a step with no testCases policy -
// simply has none.
func (p *JUnitPostProcessor) verdictFor(absPath string) (testresults.ReportVerdict, bool) {
	if len(p.verdicts) == 0 {
		return testresults.ReportVerdict{}, false
	}
	verdict, ok := p.verdicts[filepath.Clean(absPath)]
	return verdict, ok
}

// sendJUnitReport sends the JUnit report to the Agent gRPC API.
func (p *JUnitPostProcessor) sendJUnitReport(path string, report []byte, digest *testresults.Digest) error {
	// TODO: think if it's valid for the parallel steps that have independent refs
	return p.client.AppendExecutionReport(context.Background(), p.environmentId, p.executionId, p.workflowName, p.stepRef, path, report, digest)
}

// isXMLFile checks if the file is an XML file based on the extension.
func isXMLFile(stat fs.FileInfo) bool {
	if stat.IsDir() || stat.Size() == 0 {
		return false
	}

	return strings.HasSuffix(stat.Name(), ".xml")
}

func (p *JUnitPostProcessor) End() error {
	return nil
}
