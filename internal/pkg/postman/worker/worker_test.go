package worker

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorker_Run(t *testing.T) {
	// TODO implement me
	t.Skip("not implemented")
}

type RunnerMock struct {
	Error  error
	Result string
	T      *testing.T
}

func (r RunnerMock) Run(input io.Reader) (string, error) {
	body, err := ioutil.ReadAll(input)
	require.NoError(r.T, err)
	require.Contains(r.T, string(body), "kubtestExampleCollection")
	return r.Result, r.Error
}
