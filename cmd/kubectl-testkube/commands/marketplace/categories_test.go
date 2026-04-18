package marketplace

import (
	"reflect"
	"testing"

	"github.com/kubeshop/testkube/pkg/marketplace"
)

func TestAggregateCategories(t *testing.T) {
	tests := []struct {
		name      string
		workflows []marketplace.Workflow
		want      []categorySummary
	}{
		{
			name:      "empty catalog returns empty non-nil slice",
			workflows: nil,
			want:      []categorySummary{},
		},
		{
			name: "categories sorted alphabetically and components deduped",
			workflows: []marketplace.Workflow{
				{Name: "kafka-a", Category: "messaging", Component: "kafka"},
				{Name: "postgres-a", Category: "databases", Component: "postgresql"},
				{Name: "postgres-b", Category: "databases", Component: "postgresql"},
				{Name: "redis-a", Category: "databases", Component: "redis"},
				{Name: "kafka-b", Category: "messaging", Component: "kafka"},
				{Name: "nats-a", Category: "messaging", Component: "nats"},
			},
			want: []categorySummary{
				{
					Category:   "databases",
					Count:      3,
					Components: []string{"postgresql", "redis"},
				},
				{
					Category:   "messaging",
					Count:      3,
					Components: []string{"kafka", "nats"},
				},
			},
		},
		{
			name: "missing component is skipped from component list",
			workflows: []marketplace.Workflow{
				{Name: "generic", Category: "misc", Component: ""},
				{Name: "tool", Category: "misc", Component: "curl"},
			},
			want: []categorySummary{
				{
					Category:   "misc",
					Count:      2,
					Components: []string{"curl"},
				},
			},
		},
		{
			name: "single workflow yields single summary",
			workflows: []marketplace.Workflow{
				{Name: "only", Category: "networking", Component: "nginx"},
			},
			want: []categorySummary{
				{
					Category:   "networking",
					Count:      1,
					Components: []string{"nginx"},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := aggregateCategories(tc.workflows)
			if got == nil {
				t.Fatalf("aggregateCategories returned nil; want non-nil slice")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("aggregateCategories mismatch:\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestCategoriesTable_Table(t *testing.T) {
	table := categoriesTable{
		{Category: "databases", Count: 2, Components: []string{"postgresql", "redis"}},
		{Category: "messaging", Count: 1, Components: []string{"kafka"}},
	}
	header, rows := table.Table()
	wantHeader := []string{"CATEGORY", "WORKFLOWS", "COMPONENTS"}
	if !reflect.DeepEqual(header, wantHeader) {
		t.Fatalf("header mismatch: got %v want %v", header, wantHeader)
	}
	wantRows := [][]string{
		{"databases", "2", "postgresql, redis"},
		{"messaging", "1", "kafka"},
	}
	if !reflect.DeepEqual(rows, wantRows) {
		t.Fatalf("rows mismatch:\n got: %#v\nwant: %#v", rows, wantRows)
	}
}
