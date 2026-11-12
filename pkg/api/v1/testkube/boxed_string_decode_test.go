package testkube

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBoxedStringUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *BoxedString
		wantErr bool
	}{
		{name: "the object form decodes", input: `{"parallelism":{"value":"config.PARALLELISM"}}`, want: &BoxedString{Value: "config.PARALLELISM"}},
		{name: "a stored number decodes", input: `{"parallelism":2}`, want: &BoxedString{Value: "2"}},
		{name: "null stays unset", input: `{"parallelism":null}`, want: nil},
		{name: "a bare word is an error", input: `{"parallelism":"config.X"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TestWorkflowStepParallel
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Parallelism)
		})
	}
}

func TestBoxedStringRoundTripJSON(t *testing.T) {
	in := TestWorkflowStepParallel{Parallelism: &BoxedString{Value: "3"}}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `{"parallelism":{"value":"3"}}`, string(data))

	var out TestWorkflowStepParallel
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in.Parallelism, out.Parallelism)
}

func TestBoxedStringUnmarshalBSON(t *testing.T) {
	tests := []struct {
		name  string
		input bson.M
		want  *BoxedString
	}{
		{name: "the embedded document decodes", input: bson.M{"parallelism": bson.M{"value": "config.PARALLELISM"}}, want: &BoxedString{Value: "config.PARALLELISM"}},
		{name: "a stored int32 decodes", input: bson.M{"parallelism": int32(2)}, want: &BoxedString{Value: "2"}},
		{name: "a stored int64 decodes", input: bson.M{"parallelism": int64(4)}, want: &BoxedString{Value: "4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := bson.Marshal(tt.input)
			require.NoError(t, err)
			var got TestWorkflowStepParallel
			require.NoError(t, bson.Unmarshal(data, &got))
			assert.Equal(t, tt.want, got.Parallelism)
		})
	}
}
