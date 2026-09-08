package executiondata

import (
	"context"
	"io"
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

// StreamArtifact exists because read_artifact() cannot serve a report: its
// MaxInlineArtifactSize is a megabyte, smaller than the reports worth reading.
func TestStreamArtifact(t *testing.T) {
	body := strings.Repeat("x", MaxInlineArtifactSize+1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	reader, err := StreamArtifact(context.Background(), server.Client(), Artifact{
		Path: "reports/out.xml",
		Url:  server.URL,
		Size: int64(len(body)),
	})
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Len(t, content, len(body), "a report over the inline limit still streams")
}

func TestStreamArtifactRejectsAnOversizedArtifact(t *testing.T) {
	_, err := StreamArtifact(context.Background(), nil, Artifact{
		Path: "reports/huge.xml",
		Size: MaxParsedArtifactSize + 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "over the")
}

// The reported size is metadata, so the reader is capped regardless: a control
// plane understating a size must not let an unbounded body into the pod.
func TestStreamArtifactCapsTheBodyWhateverTheSizeSaid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Far more than the cap, offered by a server that claimed nothing.
		for i := 0; i < (MaxParsedArtifactSize/1024)+64; i++ {
			if _, err := w.Write([]byte(strings.Repeat("y", 1024))); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	reader, err := StreamArtifact(context.Background(), server.Client(), Artifact{
		Path: "reports/lying.xml",
		Url:  server.URL,
		Size: 0,
	})
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(content), MaxParsedArtifactSize)
}
