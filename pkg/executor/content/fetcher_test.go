//go:build integration

// this test need git to test fetchers
package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

// this content is also saved in test repo
// in https://github.com/kubeshop/testkube-examples/blob/main/example.json
// file with \n on end
const fileContent = `{"some":"json","file":"with content"}
`

func TestFetcher(t *testing.T) {
	f := NewFetcher("")

	t.Run("test fetch uri", func(t *testing.T) {

		content := &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  fileContent,
		}

		path, err := f.Fetch(content)
		assert.NoError(t, err)
		assert.FileExists(t, path)

		d, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, string(fileContent), string(d))
	})

	t.Run("test fetch git repo based file", func(t *testing.T) {
		content := &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitFile),
			Repository: testkube.NewGitRepository("https://github.com/kubeshop/testkube-examples.git", "main").
				WithPath("example.json"),
		}

		path, err := f.Fetch(content)
		assert.NoError(t, err)
		assert.FileExists(t, path)

		d, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, string(fileContent), string(d))
	})

	t.Run("test fetch git root dir", func(t *testing.T) {
		content := &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: testkube.NewGitRepository("https://github.com/kubeshop/testkube-examples.git", "main").
				WithPath(""),
		}

		path, err := f.Fetch(content)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(path, "example.json"))
		assert.FileExists(t, filepath.Join(path, "README.md"))
		assert.FileExists(t, filepath.Join(path, "subdir/example.json"))

	})

	t.Run("test fetch git subdir", func(t *testing.T) {
		content := &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: testkube.NewGitRepository("https://github.com/kubeshop/testkube-examples.git", "main").
				WithPath("subdir"),
		}

		path, err := f.Fetch(content)
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(path, "example.json"))
		assert.FileExists(t, filepath.Join(path, "example2.json"))
	})

	t.Run("test fetch no content", func(t *testing.T) {
		content := &testkube.TestContent{
			Type_: string(testkube.TestContentTypeEmpty),
		}

		path, err := f.Fetch(content)
		assert.NoError(t, err)
		assert.Equal(t, "", path)
	})
}
