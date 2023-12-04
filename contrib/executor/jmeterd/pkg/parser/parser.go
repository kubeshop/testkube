package parser

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var ErrEmptyReport = errors.New("empty JTL report")

func ParseJTLReport(report io.Reader, resultOutput []byte) (testkube.ExecutionResult, error) {
	var buf bytes.Buffer
	tee := io.TeeReader(report, &buf)
	xml := isXML(tee)
	report = io.MultiReader(&buf, report)
	if xml {
		return parseXMLReport(report, resultOutput)
	}

	return parseCSVReport(report, resultOutput)
}

func isXML(r io.Reader) bool {
	scanner := bufio.NewScanner(r)
	var firstLine string
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			firstLine = trimmedLine
			break
		}
	}

	return strings.HasPrefix(strings.TrimSpace(firstLine), "<?xml") || strings.HasPrefix(strings.TrimSpace(firstLine), "<testResults")
}

func parseXMLReport(report io.Reader, resultOutput []byte) (testkube.ExecutionResult, error) {
	data, err := io.ReadAll(report)
	if err != nil {
		return testkube.ExecutionResult{}, errors.Wrap(err, "error reading jtl report")
	}

	xmlResults, err := parseXML(data)
	if err != nil {
		return testkube.ExecutionResult{}, errors.Wrap(err, "error parsing xml jtl report")
	}

	if xmlResults.Samples == nil && xmlResults.HTTPSamples == nil {
		return testkube.ExecutionResult{}, errors.WithStack(ErrEmptyReport)
	}

	return mapXMLResultsToExecutionResults(resultOutput, xmlResults), nil
}

func parseCSVReport(report io.Reader, resultOutput []byte) (testkube.ExecutionResult, error) {
	csvResults, err := parseCSV(report)
	if err != nil {
		return testkube.ExecutionResult{}, errors.Wrap(err, "error parsing csv jtl report")
	}

	if len(csvResults.Results) == 0 {
		return testkube.ExecutionResult{}, errors.WithStack(ErrEmptyReport)
	}

	return mapCSVResultsToExecutionResults(resultOutput, csvResults), nil
}
