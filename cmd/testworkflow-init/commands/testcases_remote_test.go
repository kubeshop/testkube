package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
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
