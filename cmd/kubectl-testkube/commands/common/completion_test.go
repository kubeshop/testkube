package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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
		"workflows": CompleteTestWorkflowNames,
		"templates": CompleteTestWorkflowTemplateNames,
	} {
		t.Run(name, func(t *testing.T) {
			got, directive := fn(&cobra.Command{}, []string{"smoke"}, "")
			assert.Empty(t, got)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}
