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
