package common

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	tc "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// completionTimeout bounds how long a tab press waits for the API before it
// gives up and offers nothing.
const completionTimeout = 3 * time.Second

// nameLister returns the names of one kind of resource. It should give up
// when ctx is done.
type nameLister func(ctx context.Context) ([]string, error)

// CompleteTestWorkflowNames completes the first positional argument with the
// names of the test workflows of the current context.
func CompleteTestWorkflowNames(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeNames(commandContext(cmd), toComplete, completionTimeout, testWorkflowNames(cmd))
}

// CompleteTestWorkflowNameFlag completes a flag that takes a test workflow
// name, whatever positional arguments are already set.
func CompleteTestWorkflowNameFlag(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return completeNames(commandContext(cmd), toComplete, completionTimeout, testWorkflowNames(cmd))
}

func testWorkflowNames(cmd *cobra.Command) nameLister {
	return func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		workflows, err := c.ListTestWorkflows("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(workflows))
		for _, w := range workflows {
			names = append(names, w.Name)
		}
		return names, nil
	}
}

// CompleteTestWorkflowTemplateNames completes the first positional argument
// with the names of the test workflow templates of the current context.
func CompleteTestWorkflowTemplateNames(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeNames(commandContext(cmd), toComplete, completionTimeout, func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		templates, err := c.ListTestWorkflowTemplates("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(templates))
		for _, t := range templates {
			names = append(names, t.Name)
		}
		return names, nil
	})
}

// firstArgCompletion returns a completion function that completes the first
// positional argument with the names returned by list.
func firstArgCompletion(list func(cmd *cobra.Command) nameLister) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeNames(commandContext(cmd), toComplete, completionTimeout, list(cmd))
	}
}

// CompleteTestTriggerNames completes the first positional argument with the
// names of the test triggers.
var CompleteTestTriggerNames = firstArgCompletion(func(cmd *cobra.Command) nameLister {
	return func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		triggers, err := c.ListTestTriggers("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(triggers))
		for _, t := range triggers {
			names = append(names, t.Name)
		}
		return names, nil
	}
})

// CompleteWorkflowTriggerNames completes the first positional argument with
// the names of the workflow triggers.
var CompleteWorkflowTriggerNames = firstArgCompletion(func(cmd *cobra.Command) nameLister {
	return func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		triggers, err := c.ListWorkflowTriggers("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(triggers))
		for _, t := range triggers {
			names = append(names, t.Name)
		}
		return names, nil
	}
})

// CompleteWebhookNames completes the first positional argument with the names
// of the webhooks.
var CompleteWebhookNames = firstArgCompletion(func(cmd *cobra.Command) nameLister {
	return func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		webhooks, err := c.ListWebhooks("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(webhooks))
		for _, w := range webhooks {
			names = append(names, w.Name)
		}
		return names, nil
	}
})

// CompleteWebhookTemplateNames completes the first positional argument with
// the names of the webhook templates.
var CompleteWebhookTemplateNames = firstArgCompletion(func(cmd *cobra.Command) nameLister {
	return func(ctx context.Context) ([]string, error) {
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		templates, err := c.ListWebhookTemplates("")
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(templates))
		for _, t := range templates {
			names = append(names, t.Name)
		}
		return names, nil
	}
})

// executionCompletionLimit is how many of the most recent executions are
// offered.
const executionCompletionLimit = 50

// executionEntry is an execution offered for completion, with a short
// description for the shells that can show one.
type executionEntry struct {
	name        string
	description string
}

type executionLister func(ctx context.Context) ([]executionEntry, error)

// CompleteTestWorkflowExecutionNames completes the first positional argument
// with the names of the most recent test workflow executions, newest first.
// If the command has a --testworkflow flag that is set, only that workflow's
// executions are offered.
func CompleteTestWorkflowExecutionNames(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeExecutions(commandContext(cmd), toComplete, completionTimeout, testWorkflowExecutions(cmd, false))
}

// CompleteActiveTestWorkflowExecutionNames is like
// CompleteTestWorkflowExecutionNames, but leaves out the executions that
// already finished, for commands such as cancel.
func CompleteActiveTestWorkflowExecutionNames(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeExecutions(commandContext(cmd), toComplete, completionTimeout, testWorkflowExecutions(cmd, true))
}

func testWorkflowExecutions(cmd *cobra.Command, activeOnly bool) executionLister {
	return func(ctx context.Context) ([]executionEntry, error) {
		workflow := ""
		if f := cmd.Flags().Lookup("testworkflow"); f != nil {
			workflow = f.Value.String()
		}
		c, _, err := getClient(ctx, cmd, false)
		if err != nil {
			return nil, err
		}
		result, err := c.ListTestWorkflowExecutions(workflow, executionCompletionLimit, tc.FilterTestWorkflowExecutionOptions{})
		if err != nil {
			return nil, err
		}
		return executionEntries(result.Results, activeOnly), nil
	}
}

// executionEntries turns execution summaries into completion entries and
// keeps their order. Executions without a name are offered by ID, which the
// commands accept too.
func executionEntries(executions []testkube.TestWorkflowExecutionSummary, activeOnly bool) []executionEntry {
	entries := make([]executionEntry, 0, len(executions))
	for _, e := range executions {
		var status testkube.TestWorkflowStatus
		if e.Result != nil && e.Result.Status != nil {
			status = *e.Result.Status
		}
		if activeOnly && status != "" && status.Finished() {
			continue
		}
		name := e.Name
		if name == "" {
			name = e.Id
		}
		var parts []string
		if e.Workflow != nil && e.Workflow.Name != "" {
			parts = append(parts, e.Workflow.Name)
		}
		if status != "" {
			parts = append(parts, string(status))
		}
		entries = append(entries, executionEntry{name: name, description: strings.Join(parts, ", ")})
	}
	return entries
}

// completeExecutions offers the executions list returns whose name starts
// with toComplete, in the order list returns them. Failures offer nothing, as
// in completeNames.
func completeExecutions(ctx context.Context, toComplete string, timeout time.Duration, list executionLister) ([]cobra.Completion, cobra.ShellCompDirective) {
	const directive = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	entries, ok := runWithTimeout(ctx, timeout, list)
	if !ok {
		return nil, directive
	}
	var out []cobra.Completion
	for _, e := range entries {
		if e.name == "" || !strings.HasPrefix(e.name, toComplete) {
			continue
		}
		if e.description != "" {
			out = append(out, cobra.CompletionWithDesc(e.name, e.description))
		} else {
			out = append(out, e.name)
		}
	}
	return out, directive
}

// completeNames offers the names list returns that start with toComplete.
// Completion runs on every tab press, so any failure (no config, no cluster,
// expired login, timeout) offers nothing instead of printing, prompting or
// waiting.
func completeNames(ctx context.Context, toComplete string, timeout time.Duration, list nameLister) ([]cobra.Completion, cobra.ShellCompDirective) {
	names, ok := runWithTimeout(ctx, timeout, list)
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return filterByPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// runWithTimeout runs list and reports false if it fails, panics or doesn't
// return within timeout.
func runWithTimeout[T any](ctx context.Context, timeout time.Duration, list func(ctx context.Context) ([]T, error)) ([]T, bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		items []T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("listing names: %v", r)}
			}
		}()
		items, err := list(ctx)
		done <- result{items: items, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, false
	case r := <-done:
		if r.err != nil {
			return nil, false
		}
		return r.items, true
	}
}

func filterByPrefix(names []string, prefix string) []cobra.Completion {
	var out []cobra.Completion
	for _, name := range names {
		if name != "" && strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func commandContext(cmd *cobra.Command) context.Context {
	if cmd != nil && cmd.Context() != nil {
		return cmd.Context()
	}
	return context.Background()
}
