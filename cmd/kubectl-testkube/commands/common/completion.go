package common

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
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

// completeNames offers the names list returns that start with toComplete.
// Completion runs on every tab press, so any failure (no config, no cluster,
// expired login, timeout) offers nothing instead of printing, prompting or
// waiting.
func completeNames(ctx context.Context, toComplete string, timeout time.Duration, list nameLister) ([]cobra.Completion, cobra.ShellCompDirective) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		names []string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("listing names: %v", r)}
			}
		}()
		names, err := list(ctx)
		done <- result{names: names, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, cobra.ShellCompDirectiveNoFileComp
	case r := <-done:
		if r.err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return filterByPrefix(r.names, toComplete), cobra.ShellCompDirectiveNoFileComp
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
