package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// remoteReport stands up an execution whose report is served from object
// storage, which is how another execution's report is actually reached: the
// control plane hands back a presigned URL and the pod streams it.
func remoteReport(t *testing.T, executionId, body string) (reportSource, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), executionId).
		Return(executiondata.Execution{Id: executionId}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), executionId, gomock.Any()).
		Return([]executiondata.Artifact{{
			Path: "reports/out.xml",
			Url:  server.URL + "/reports/out.xml",
			Size: int64(len(body)),
		}}, nil).
		AnyTimes()

	return reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: executionId},
		Repository: repository,
		Client:     server.Client(),
	}, server
}

// The whole point of --only-failed: the first attempt of a rerun has no report
// of its own, so it takes the failures of the execution being rerun. Before
// this, the reference reached the pod and was ignored, and the suite ran whole.
func TestResolveSelection_SeedsFromTheRerunExecution(t *testing.T) {
	source, _ := remoteReport(t, "exec-1", previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", As: `{{ testcase.name }}`},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1", OnlyFailed: true}, source)
	require.NoError(t, err)

	assert.True(t, selection.Narrowed)
	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries,
		"the passing case is not re-run")
	assert.Equal(t, []string{
		"s/tests.a/test_flaky_a", "s/tests.b/test_real_bug", "s/tests.b/test_boom",
	}, selection.Addresses, "the addresses come across too, so the verdict can check them")
}

// The step's own report wins once it has one, which is what keeps a narrowing
// retry narrowing: attempt 2 takes attempt 1's failures, not the original
// execution's. That ordering is why neither source needs configuring.
func TestResolveSelection_PrefersItsOwnReportOverTheRerun(t *testing.T) {
	source, _ := remoteReport(t, "exec-1", previousAttempt)
	dir := reportWith(t, `<testsuite name="s" tests="2">
  <testcase name="test_ok" classname="tests.a"/>
  <testcase name="test_local_only" classname="tests.a"><failure message="broken"/></testcase>
</testsuite>`)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", As: `{{ testcase.name }}`},
	}, dir, &testworkflowconfig.RerunConfig{ExecutionId: "exec-1", OnlyFailed: true}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_local_only"}, selection.Entries)
}

// An explicit source reads that execution and nothing else - no local fallback,
// because the workflow named where the results come from.
func TestResolveSelection_ReadsAnExplicitRemoteSource(t *testing.T) {
	source, _ := remoteReport(t, "exec-1", previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/second.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries)
}

// A reference naming an execution that does not exist is a mistake in the
// workflow, so it fails rather than quietly selecting nothing and running the
// whole suite.
func TestResolveSelection_UnresolvableSourceFails(t *testing.T) {
	// A pod running an execution that is not a rerun has no id for the reference
	// to resolve to.
	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	source := reportSource{Resolver: executiondata.Resolver{Repository: repository}, Repository: repository}

	_, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
		},
	}, t.TempDir(), nil, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a rerun of another one",
		"the reference has to say why it could not resolve")
}

// The other execution having produced no matching report is not an error: it is
// the same "nothing to narrow to" as a missing local one, and `empty` decides.
func TestResolveSelection_RemoteSourceWithNoReport(t *testing.T) {
	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), "exec-1").
		Return(executiondata.Execution{Id: "exec-1"}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
		Return(nil, nil).
		AnyTimes()

	source := reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: "exec-1"},
		Repository: repository,
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)
	assert.False(t, selection.Narrowed)
}

// recordedFailures builds a source whose execution record already carries the
// failures, and whose artifact store would fail if touched - the point being
// that it is not touched.
func recordedFailures(t *testing.T, executionId string, reports []executiondata.Report) reportSource {
	t.Helper()

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), executionId).
		Return(executiondata.Execution{Id: executionId, Reports: reports}, nil).
		AnyTimes()

	return reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: executionId},
		Repository: repository,
	}
}

func failed(ids ...string) []executiondata.ReportFailure {
	failures := make([]executiondata.ReportFailure, 0, len(ids))
	for _, id := range ids {
		failures = append(failures, executiondata.ReportFailure{Id: id, Status: "failed"})
	}
	return failures
}

// The record outlives the artifacts, so preferring it is what lets a rerun
// narrow against an execution whose report file has been pruned. It also spares
// the pod a download and the deployment an artifact-read capability.
//
// ListArtifacts is deliberately not expected: gomock failing on an unexpected
// call is the assertion that nothing was downloaded.
func TestResolveSelection_PrefersTheFailuresOnTheRecord(t *testing.T) {
	source := recordedFailures(t, "exec-1", []executiondata.Report{{
		Ref:      "step1",
		File:     "reports/out.xml",
		Failures: failed("s/tests.b/test_real_bug", "s/tests.b/test_boom"),
	}})

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_real_bug", "test_boom"}, selection.Entries)
	assert.Equal(t, []string{"s/tests.b/test_real_bug", "s/tests.b/test_boom"}, selection.Addresses)
}

// The muted failures on the record are excluded the same way they are in a
// parsed report, since both go through the same selection.
func TestResolveSelection_RecordRespectsMute(t *testing.T) {
	source := recordedFailures(t, "exec-1", []executiondata.Report{{
		File: "reports/out.xml",
		Failures: []executiondata.ReportFailure{
			{Id: "s/c/test_flaky_a", Status: "failed", Muted: true},
			{Id: "s/c/test_real", Status: "failed"},
		},
	}})

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		MuteInclude: []string{"test_flaky_*"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"s/c/test_real"}, selection.Entries)
}

// A truncated list must send the caller to the report rather than narrow to its
// prefix: narrowing to the first 2000 of 5000 failures would test less than it
// claimed and pass on the rest.
func TestResolveSelection_TruncatedRecordFallsBackToTheReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(previousAttempt))
	}))
	defer server.Close()

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), "exec-1").
		Return(executiondata.Execution{Id: "exec-1", Reports: []executiondata.Report{{
			Failures:  failed("s/c/only-the-first-one"),
			Truncated: true,
		}}}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
		Return([]executiondata.Artifact{{
			Path: "reports/out.xml",
			Url:  server.URL,
			Size: int64(len(previousAttempt)),
		}}, nil)

	source := reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: "exec-1"},
		Repository: repository,
		Client:     server.Client(),
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries,
		"the whole story comes from the report, not the prefix on the record")
}

// A record carrying no reports means nothing was stored, not that nothing
// failed, so the report still has to be read.
func TestResolveSelection_RecordWithoutReportsFallsBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(previousAttempt))
	}))
	defer server.Close()

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), "exec-1").
		Return(executiondata.Execution{Id: "exec-1"}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
		Return([]executiondata.Artifact{{
			Path: "reports/out.xml",
			Url:  server.URL,
			Size: int64(len(previousAttempt)),
		}}, nil)

	source := reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: "exec-1"},
		Repository: repository,
		Client:     server.Client(),
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Len(t, selection.Entries, 3)
}

// An execution has a report per step that produced one, and they are different
// suites. Merging them all would hand the runner test cases belonging to another
// step, which either run the wrong tests or name nothing the tool recognises.
func TestResolveSelection_RecordTakesOnlyTheNamedReport(t *testing.T) {
	source := recordedFailures(t, "exec-1", []executiondata.Report{
		{
			Ref:      "unit",
			File:     "reports/unit.xml",
			Failures: failed("unit/UnitTest/test_one"),
		},
		{
			Ref:      "integration",
			File:     "reports/integration.xml",
			Failures: failed("integration/IntegrationTest/test_two"),
		},
	})

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/unit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/unit.xml"},
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"unit/UnitTest/test_one"}, selection.Entries,
		"the other step's failures belong to another suite")
}

// A glob selects the reports it names and no others.
func TestResolveSelection_RecordMatchesGlobs(t *testing.T) {
	source := recordedFailures(t, "exec-1", []executiondata.Report{
		{File: "reports/shard-1.xml", Failures: failed("s/c/test_one")},
		{File: "reports/shard-2.xml", Failures: failed("s/c/test_two")},
		{File: "other/unrelated.xml", Failures: failed("other/c/test_three")},
	})

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/*.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/*.xml"},
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"s/c/test_one", "s/c/test_two"}, selection.Entries)
}

// A report stored under an upload prefix keeps its name but not its leading
// path, and matching nothing sends the caller to the artifacts - where the
// control plane does the matching, as it does for every other artifact lookup.
func TestResolveSelection_RecordThatMatchesNothingFallsBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(previousAttempt))
	}))
	defer server.Close()

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), "exec-1").
		Return(executiondata.Execution{Id: "exec-1", Reports: []executiondata.Report{{
			File:     "chrome/junit/junit-430a259975c005f61f158f2084f5b1dc.xml",
			Failures: failed("s/c/stale"),
		}}}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
		Return([]executiondata.Artifact{{
			Path: "reports/out.xml",
			Url:  server.URL,
			Size: int64(len(previousAttempt)),
		}}, nil)

	source := reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: "exec-1"},
		Repository: repository,
		Client:     server.Client(),
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"reports/out.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"reports/out.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries,
		"the artifacts answer when the stored path does not match")
}

// Matching is against the whole stored path, with no fall back to the base
// name: a slashless pattern means a report at the root and nothing else. See
// TestResolveSelection_RecordDoesNotMergeReportsSharingABaseName for why.
func TestMatchesReportPath(t *testing.T) {
	for _, tc := range []struct {
		file     string
		patterns []string
		want     bool
	}{
		{"reports/out.xml", []string{"reports/out.xml"}, true},
		{"reports/out.xml", []string{"reports/*.xml"}, true},
		{"reports/nested/out.xml", []string{"reports/**/*.xml"}, true},
		{"reports/out.xml", []string{"reports/other.xml"}, false},
		{"junit.xml", []string{"junit.xml"}, true},
		{"unit/junit.xml", []string{"junit.xml"}, false},
		{"prefix/reports/out.xml", []string{"out.xml"}, false},
		{"prefix/reports/out.xml", []string{"reports/out.xml"}, false},
		{"prefix/reports/out.xml", []string{"**/reports/out.xml"}, true},
		{"", []string{"reports/out.xml"}, false},
		{"reports/out.xml", nil, false},
	} {
		t.Run(tc.file+" vs "+strings.Join(tc.patterns, ","), func(t *testing.T) {
			assert.Equal(t, tc.want, matchesReportPath(tc.file, tc.patterns))
		})
	}
}

// Several steps of one execution commonly upload a report of the same name
// under different prefixes. A slashless pattern must not sweep them all up:
// merging two suites hands the runner test cases from a step it knows nothing
// about, which is the whole reason the paths are matched at all.
//
// Nothing matches here, so the artifacts answer instead - where the control
// plane matches against how they are really stored.
func TestResolveSelection_RecordDoesNotMergeReportsSharingABaseName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(previousAttempt))
	}))
	defer server.Close()

	repository := executiondata.NewMockExecutionRepository(gomock.NewController(t))
	repository.EXPECT().
		Get(gomock.Any(), "exec-1").
		Return(executiondata.Execution{Id: "exec-1", Reports: []executiondata.Report{
			{Ref: "unit", File: "unit/junit.xml", Failures: failed("unit/UnitTest/test_one")},
			{Ref: "integration", File: "integration/junit.xml", Failures: failed("integration/IntegrationTest/test_two")},
		}}, nil).
		AnyTimes()
	repository.EXPECT().
		ListArtifacts(gomock.Any(), "exec-1", gomock.Any()).
		Return([]executiondata.Artifact{{
			Path: "junit.xml",
			Url:  server.URL,
			Size: int64(len(previousAttempt)),
		}}, nil)

	source := reportSource{
		Resolver:   executiondata.Resolver{Repository: repository, RerunId: "exec-1"},
		Repository: repository,
		Client:     server.Client(),
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:  executiondata.RerunRef,
			Paths: []string{"junit.xml"},
			As:    `{{ testcase.name }}`,
		},
	}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1"}, source)
	require.NoError(t, err)

	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries)
	assert.NotContains(t, selection.Entries, "test_one", "the unit step's failures are not this step's")
	assert.NotContains(t, selection.Entries, "test_two", "and neither are the integration step's")
}
