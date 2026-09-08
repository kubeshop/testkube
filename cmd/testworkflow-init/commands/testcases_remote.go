package commands

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/testresults"
)

// reportSource reads a test report belonging to another execution.
//
// It exists so that resolveTestCaseSelection can take its previous results from
// somewhere other than this pod's own file system without knowing anything about
// the control plane.
type reportSource struct {
	Resolver   executiondata.Resolver
	Repository executiondata.ExecutionRepository
	Client     *http.Client
}

// readRemoteReport resolves ref to an execution and parses the reports matching
// patterns out of its artifacts.
//
// Finding the execution but none of its reports is not an error, the same way a
// missing local report is not: it yields an empty report and the caller's `empty`
// policy decides. Failing to resolve the reference *is* an error - a reference
// that names nothing is a mistake in the workflow, not an empty result.
func (s reportSource) readRemoteReport(ctx context.Context, ref string, patterns []string) (testresults.Report, bool, error) {
	if s.Repository == nil {
		return testresults.Report{}, false, fmt.Errorf(
			"select.from is %q, which reads another execution's report, and this workflow has no connection to the control plane", ref)
	}
	if len(patterns) == 0 {
		return testresults.Report{}, false, fmt.Errorf(
			"select.from is %q, so select.paths has to say which of its artifacts hold the report", ref)
	}

	execution, err := s.Resolver.Resolve(ctx, ref, 0)
	if err != nil {
		return testresults.Report{}, false, err
	}

	artifacts, err := s.Repository.ListArtifacts(ctx, execution.Id, patterns)
	if err != nil {
		return testresults.Report{}, false, fmt.Errorf("listing the reports of execution %s: %w", execution.Id, err)
	}
	if len(artifacts) == 0 {
		return testresults.Report{}, false, nil
	}

	reports := make([]testresults.Report, 0, len(artifacts))
	for _, artifact := range artifacts {
		report, err := s.parseArtifact(ctx, artifact)
		if err != nil {
			return testresults.Report{}, false, err
		}
		reports = append(reports, report)
	}
	return testresults.Merge(reports...), true, nil
}

// parseArtifact streams one artifact through the parser, so that a report of
// several megabytes never lands in memory whole.
func (s reportSource) parseArtifact(ctx context.Context, artifact executiondata.Artifact) (testresults.Report, error) {
	body, err := executiondata.StreamArtifact(ctx, s.Client, artifact)
	if err != nil {
		return testresults.Report{}, err
	}
	defer body.Close()

	report, err := testresults.Parse(body)
	if err != nil {
		return testresults.Report{}, fmt.Errorf("parsing report %q: %w", artifact.Path, err)
	}
	return report, nil
}

// executionReportSource wires the report reader to the control plane the way an
// expression's execution() reference is wired, so that `from: parent` and
// `execution("parent")` mean the same execution.
func executionReportSource() reportSource {
	return reportSource{
		Resolver:   data.ExecutionResolver(data.ExecutionRegistry()),
		Repository: data.ExecutionDataRepository(),
		Client:     data.ArtifactClient(),
	}
}
