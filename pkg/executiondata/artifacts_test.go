package executiondata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestReadArtifactFunction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/summary.json":
			_, _ = w.Write([]byte(`{"passed":true}`))
		case "/huge":
			_, _ = w.Write([]byte(strings.Repeat("x", MaxInlineArtifactSize+1)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Add(Execution{Id: "exec-1", Workflow: "producer", Alias: "p"})

	t.Run("returns the artifact content", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"results/summary.json"}).
			Return([]Artifact{{Path: "results/summary.json", Url: server.URL + "/summary.json", Size: 15}}, nil)

		machine := NewMachine(MachineOptions{Registry: registry, Repository: repository})
		value, err := resolve(t, `read_artifact("p", "results/summary.json")`, machine)
		require.NoError(t, err)
		assert.Equal(t, `{"passed":true}`, value)
	})

	t.Run("points at fetch for oversized artifacts", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"big.bin"}).
			Return([]Artifact{{Path: "big.bin", Url: server.URL + "/huge", Size: MaxInlineArtifactSize + 1}}, nil)

		machine := NewMachine(MachineOptions{Registry: registry, Repository: repository})
		_, err := resolve(t, `read_artifact("p", "big.bin")`, machine)
		assert.ErrorContains(t, err, "use a 'fetch' block")
	})

	t.Run("rejects an artifact the control plane reported no size for", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"big.bin"}).
			Return([]Artifact{{Path: "big.bin", Url: server.URL + "/huge"}}, nil)

		machine := NewMachine(MachineOptions{Registry: registry, Repository: repository})
		_, err := resolve(t, `read_artifact("p", "big.bin")`, machine)
		assert.ErrorContains(t, err, "use a 'fetch' block")
	})

	t.Run("reports a missing artifact", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"nope.txt"}).Return(nil, nil)

		machine := NewMachine(MachineOptions{Registry: registry, Repository: repository})
		_, err := resolve(t, `read_artifact("p", "nope.txt")`, machine)
		assert.ErrorContains(t, err, `artifact "nope.txt" not found`)
	})

	t.Run("refuses an ambiguous pattern", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().ListArtifacts(gomock.Any(), "exec-1", []string{"results/*"}).
			Return([]Artifact{{Path: "results/a.txt"}, {Path: "results/b.txt"}}, nil)

		machine := NewMachine(MachineOptions{Registry: registry, Repository: repository})
		_, err := resolve(t, `read_artifact("p", "results/*")`, machine)
		assert.ErrorContains(t, err, "matches 2 artifacts")
	})

	t.Run("rejects malformed arguments", func(t *testing.T) {
		machine := NewMachine(MachineOptions{Registry: registry})

		_, err := resolve(t, `read_artifact("p")`, machine)
		assert.ErrorContains(t, err, "expects 2 arguments")

		_, err = resolve(t, `read_artifact("p", "")`, machine)
		assert.ErrorContains(t, err, "expects a non-empty path")
	})

	t.Run("explains a missing control plane connection", func(t *testing.T) {
		machine := NewMachine(MachineOptions{Registry: registry})
		_, err := resolve(t, `read_artifact("p", "results/summary.json")`, machine)
		assert.ErrorContains(t, err, "no connection to the control plane")
	})
}
