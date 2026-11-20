package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func staticNames(names ...string) nameLister {
	return func(context.Context) ([]string, error) { return names, nil }
}

func TestCompleteNames(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		list       nameLister
		want       []cobra.Completion
	}{
		{
			name: "empty prefix offers every name, sorted",
			list: staticNames("smoke", "k6-load", "api-e2e"),
			want: []cobra.Completion{"api-e2e", "k6-load", "smoke"},
		},
		{
			name:       "prefix filters names",
			toComplete: "api",
			list:       staticNames("api-e2e", "smoke", "api-contract", "cypress-api"),
			want:       []cobra.Completion{"api-contract", "api-e2e"},
		},
		{
			name:       "no match offers nothing",
			toComplete: "zzz",
			list:       staticNames("api-e2e", "smoke"),
		},
		{
			name: "empty names are dropped",
			list: staticNames("", "smoke"),
			want: []cobra.Completion{"smoke"},
		},
		{
			name: "list error offers nothing",
			list: func(context.Context) ([]string, error) {
				return []string{"smoke"}, errors.New("no cluster")
			},
		},
		{
			name: "panic in list offers nothing",
			list: func(context.Context) ([]string, error) { panic("boom") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, directive := completeNames(context.Background(), tt.toComplete, time.Second, tt.list)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestCompleteNamesGivesUpAfterTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	slow := func(context.Context) ([]string, error) {
		<-release
		return []string{"smoke"}, nil
	}

	start := time.Now()
	got, directive := completeNames(context.Background(), "", 50*time.Millisecond, slow)

	assert.Empty(t, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Less(t, time.Since(start), time.Second)
}

func TestCompleteNamesPassesDeadlineToList(t *testing.T) {
	var hasDeadline bool
	list := func(ctx context.Context) ([]string, error) {
		_, hasDeadline = ctx.Deadline()
		return nil, nil
	}

	completeNames(context.Background(), "", time.Second, list)

	assert.True(t, hasDeadline)
}

func TestCompleteNamesOffersNothingOnceNameIsSet(t *testing.T) {
	// With a positional argument set, the functions return before building a
	// client, so a bare command with no flags is enough.
	for name, fn := range map[string]cobra.CompletionFunc{
		"workflows":         CompleteTestWorkflowNames,
		"templates":         CompleteTestWorkflowTemplateNames,
		"executions":        CompleteTestWorkflowExecutionNames,
		"active executions": CompleteActiveTestWorkflowExecutionNames,
		"test triggers":     CompleteTestTriggerNames,
		"workflow triggers": CompleteWorkflowTriggerNames,
		"webhooks":          CompleteWebhookNames,
		"webhook templates": CompleteWebhookTemplateNames,
	} {
		t.Run(name, func(t *testing.T) {
			got, directive := fn(&cobra.Command{}, []string{"smoke"}, "")
			assert.Empty(t, got)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive&cobra.ShellCompDirectiveNoFileComp)
		})
	}
}

func execSummary(id, name, workflow string, status testkube.TestWorkflowStatus) testkube.TestWorkflowExecutionSummary {
	e := testkube.TestWorkflowExecutionSummary{Id: id, Name: name}
	if workflow != "" {
		e.Workflow = &testkube.TestWorkflowSummary{Name: workflow}
	}
	if status != "" {
		e.Result = &testkube.TestWorkflowResultSummary{Status: &status}
	}
	return e
}

func TestExecutionEntries(t *testing.T) {
	executions := []testkube.TestWorkflowExecutionSummary{
		execSummary("id-3", "smoke-3", "smoke", testkube.RUNNING_TestWorkflowStatus),
		execSummary("id-2", "smoke-2", "smoke", testkube.PASSED_TestWorkflowStatus),
		execSummary("id-1", "", "", testkube.QUEUED_TestWorkflowStatus),
		execSummary("id-0", "k6-1", "k6", ""),
	}

	assert.Equal(t, []executionEntry{
		{name: "smoke-3", description: "smoke, running"},
		{name: "smoke-2", description: "smoke, passed"},
		{name: "id-1", description: "queued"},
		{name: "k6-1", description: "k6"},
	}, executionEntries(executions, false))

	assert.Equal(t, []executionEntry{
		{name: "smoke-3", description: "smoke, running"},
		{name: "id-1", description: "queued"},
		{name: "k6-1", description: "k6"},
	}, executionEntries(executions, true))
}

func TestCompleteExecutions(t *testing.T) {
	const directive = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	entries := func(context.Context) ([]executionEntry, error) {
		return []executionEntry{
			{name: "smoke-10", description: "smoke, running"},
			{name: "k6-4"},
			{name: "smoke-9", description: "smoke, failed"},
			{name: ""},
		}, nil
	}

	got, d := completeExecutions(context.Background(), "", time.Second, entries)
	assert.Equal(t, []cobra.Completion{"smoke-10	smoke, running", "k6-4", "smoke-9	smoke, failed"}, got)
	assert.Equal(t, directive, d)

	got, _ = completeExecutions(context.Background(), "smoke", time.Second, entries)
	assert.Equal(t, []cobra.Completion{"smoke-10	smoke, running", "smoke-9	smoke, failed"}, got)

	failing := func(context.Context) ([]executionEntry, error) { return nil, errors.New("no cluster") }
	got, d = completeExecutions(context.Background(), "", time.Second, failing)
	assert.Empty(t, got)
	assert.Equal(t, directive, d)
}
