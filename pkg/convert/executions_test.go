package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// rawDoc marshals a document the way the driver hands it to the cursor.
func rawDoc(t *testing.T, doc any) bson.Raw {
	t.Helper()
	b, err := bson.Marshal(doc)
	require.NoError(t, err)
	return bson.Raw(b)
}

func TestDocumentMongoID(t *testing.T) {
	t.Parallel()

	t.Run("reads the id without decoding the rest", func(t *testing.T) {
		t.Parallel()

		want := bson.NewObjectID()
		raw := rawDoc(t, bson.D{
			{Key: "_id", Value: want},
			{Key: "id", Value: "exec-1"},
			{Key: "name", Value: "wf-run-1"},
		})

		got, err := documentMongoID(raw)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	// The execution's own id field is a separate string column. Confusing the two
	// would checkpoint a position that _id paging cannot use.
	t.Run("does not confuse _id with the execution id", func(t *testing.T) {
		t.Parallel()

		want := bson.NewObjectID()
		raw := rawDoc(t, bson.D{
			{Key: "id", Value: "exec-1"},
			{Key: "_id", Value: want},
		})

		got, err := documentMongoID(raw)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("rejects a document with no _id", func(t *testing.T) {
		t.Parallel()

		raw := rawDoc(t, bson.D{{Key: "id", Value: "exec-1"}})

		_, err := documentMongoID(raw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no _id")
	})

	// The checkpoint stores the position as a hex ObjectID, so an _id of any other
	// type cannot be represented and must not be silently accepted.
	t.Run("rejects a non-ObjectID _id", func(t *testing.T) {
		t.Parallel()

		for name, doc := range map[string]bson.D{
			"string": {{Key: "_id", Value: "custom-id"}},
			"int":    {{Key: "_id", Value: int64(42)}},
			"subdoc": {{Key: "_id", Value: bson.D{{Key: "a", Value: 1}}}},
			"null":   {{Key: "_id", Value: nil}},
		} {
			_, err := documentMongoID(rawDoc(t, doc))
			require.Error(t, err, "an _id of type %s must be rejected", name)
			assert.Contains(t, err.Error(), "not an ObjectID")
		}
	})

	t.Run("finds _id even when it is not the first field", func(t *testing.T) {
		t.Parallel()

		want := bson.NewObjectID()
		raw := rawDoc(t, bson.D{
			{Key: "name", Value: "wf-run-1"},
			{Key: "workflow", Value: bson.D{{Key: "name", Value: "wf"}}},
			{Key: "_id", Value: want},
		})

		got, err := documentMongoID(raw)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

// The integration fixture is written through the Mongo repository, whose Insert
// calls EscapeDots. TestWorkflow.ConvertDots guards a nil workflow but then
// dereferences its Spec unguarded, so a fixture missing a spec panics inside the
// repository rather than failing an assertion.
//
// This runs without a database on purpose: the fixture's compatibility with the
// repository write path is exactly the kind of breakage that should not have to
// wait for the integration job to surface it.
func TestBuildExecutionSurvivesTheRepositoryWritePath(t *testing.T) {
	t.Parallel()

	execution := buildExecution(1)

	require.NotPanics(t, func() {
		execution.Clone().EscapeDots()
	}, "the fixture must survive the escaping the Mongo repository applies on insert")

	require.NotPanics(t, func() {
		execution.Clone().UnscapeDots()
	}, "and the unescaping both repositories apply on read")

	// Every workflow snapshot needs a spec, since ConvertDots reaches through it.
	require.NotNil(t, execution.Workflow, "the fixture must carry a workflow snapshot")
	assert.NotNil(t, execution.Workflow.Spec, "the workflow snapshot must carry a spec")
	require.NotNil(t, execution.ResolvedWorkflow, "and a resolved snapshot")
	assert.NotNil(t, execution.ResolvedWorkflow.Spec, "the resolved snapshot must carry a spec")
}

// Escaping is reversible, so a document written by the repository and read back
// gives the original keys. The migrator relies on this by copying stored
// documents through untouched.
func TestFixtureDotEscapingRoundTrips(t *testing.T) {
	t.Parallel()

	original := buildExecution(1)
	roundTripped := original.Clone().EscapeDots().UnscapeDots()

	assert.Equal(t, original.Tags, roundTripped.Tags)
	assert.Equal(t, original.Workflow.Labels, roundTripped.Workflow.Labels)
	assert.Equal(t, original.Workflow.Spec.Pod.Labels, roundTripped.Workflow.Spec.Pod.Labels)

	// And the escaped form really is different, or the assertions above would
	// pass on a fixture with no dotted keys at all.
	escaped := original.Clone().EscapeDots()
	assert.NotEqual(t, original.Tags, escaped.Tags,
		"the fixture must contain dotted keys for this to be testing anything")
}
