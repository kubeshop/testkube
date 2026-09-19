package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/repository/sequence"
)

func testSequenceMigrator() *sequenceMigrator {
	return &sequenceMigrator{log: zap.NewNop().Sugar()}
}

func TestResolveSequenceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      sequenceDoc
		wantName string
		wantOK   bool
		reason   string
	}{
		{
			name:     "test workflow counter",
			doc:      sequenceDoc{ID: "tw-my-workflow", Number: 17, ExecutionType: sequence.ExecutionTypeTestWorkflow},
			wantName: "my-workflow",
			wantOK:   true,
		},
		{
			name:     "workflow name containing a hyphen",
			doc:      sequenceDoc{ID: "tw-my-long-workflow-name", ExecutionType: sequence.ExecutionTypeTestWorkflow},
			wantName: "my-long-workflow-name",
			wantOK:   true,
			reason:   "only the first prefix is stripped, not every hyphen",
		},
		{
			name:   "legacy test counter is dropped",
			doc:    sequenceDoc{ID: "t-my-test", ExecutionType: sequence.ExecutionTypeTest},
			wantOK: false,
			reason: "Tests have no Postgres repository, and the target keyspace ignores the type",
		},
		{
			name:   "legacy test suite counter is dropped",
			doc:    sequenceDoc{ID: "ts-my-suite", ExecutionType: sequence.ExecutionTypeTestSuite},
			wantOK: false,
		},
		{
			name:   "mismatched prefix is dropped",
			doc:    sequenceDoc{ID: "my-workflow", ExecutionType: sequence.ExecutionTypeTestWorkflow},
			wantOK: false,
			reason: "the id must actually carry the type prefix it claims",
		},
		{
			name:   "empty name after the prefix is dropped",
			doc:    sequenceDoc{ID: "tw-", ExecutionType: sequence.ExecutionTypeTestWorkflow},
			wantOK: false,
			reason: "name is part of the primary key and cannot be empty",
		},
		{
			name:     "legacy name-keyed document is a test workflow counter",
			doc:      sequenceDoc{LegacyName: "my-workflow", Number: 5},
			wantName: "my-workflow",
			wantOK:   true,
			reason:   "pre-executionType documents carried an unprefixed name",
		},
		{
			name:   "legacy name-keyed test suite counter is dropped",
			doc:    sequenceDoc{LegacyName: "ts-my-suite"},
			wantOK: false,
			reason: "the ts- prefix marked a test suite in the legacy shape",
		},
		{
			name:   "document with neither type nor name is dropped",
			doc:    sequenceDoc{ID: "something"},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := newStats(TaskSequences)
			name, ok := testSequenceMigrator().resolveName(&tt.doc, stats)

			assert.Equal(t, tt.wantOK, ok, tt.reason)
			assert.Equal(t, tt.wantName, name, tt.reason)

			if tt.wantOK {
				assert.Zero(t, stats.Skipped, "a migratable document must not be counted as skipped")
			} else {
				assert.Equal(t, int64(1), stats.Skipped,
					"a declined document must be counted as skipped, not failed")
				assert.Zero(t, stats.Failed, "declining a document is not a failure")
			}
		})
	}
}

// The migrated counter must never be lower than what Mongo held, or executions
// scheduled after the cutover would reuse the numbers of migrated ones. The
// upsert applies GREATEST in SQL; this covers the in-memory fold that feeds it.
func TestSequenceFoldKeepsHighestNumber(t *testing.T) {
	t.Parallel()

	m := testSequenceMigrator()
	stats := newStats(TaskSequences)

	docs := []sequenceDoc{
		// Both shapes can describe the same workflow: a legacy name-keyed
		// document and the current id-keyed one.
		{LegacyName: "my-workflow", Number: 12},
		{ID: "tw-my-workflow", Number: 40, ExecutionType: sequence.ExecutionTypeTestWorkflow},
		{ID: "tw-other", Number: 3, ExecutionType: sequence.ExecutionTypeTestWorkflow},
	}

	highest := make(map[string]int32)
	for i := range docs {
		name, ok := m.resolveName(&docs[i], stats)
		if !ok {
			continue
		}
		if current, seen := highest[name]; !seen || docs[i].Number > current {
			highest[name] = docs[i].Number
		}
	}

	assert.Equal(t, map[string]int32{"my-workflow": 40, "other": 3}, highest)
}
