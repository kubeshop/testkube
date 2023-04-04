package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTarball_Extract(t *testing.T) {
	t.Parallel()

	// create a test tarball
	var buf bytes.Buffer
	tarball := NewTarballService()
	content := "testfile\n"
	files := []*File{
		{Name: "testfile.txt", Mode: 0644, Size: 9, ModTime: time.Now(), Data: bytes.NewBufferString(content)},
		{Name: "../hack.txt", Mode: 0644, Size: 9, ModTime: time.Now(), Data: bytes.NewBufferString(content)},
	}
	if err := tarball.Create(&buf, files); err != nil {
		t.Fatalf("error creating tarball: %v", err)
	}

	files, err := tarball.Extract(&buf)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}
	assert.Equalf(t, "testfile.txt", files[0].Name, "Extract() returned file with name %s, expected testfile.txt", files[0].Name)
	assert.Equalf(t, int64(0644), files[0].Mode, "Extract() returned file with mode %o, expected 0644", files[0].Mode)
	if files[0].Mode != 0644 {
		t.Fatalf("Extract() returned file with mode %o, expected 0644", files[0].Mode)
	}
	assert.Equalf(t, int64(len(content)), files[0].Size, "Extract() returned file with size %d, expected %d", files[0].Size, len(content))
	if files[0].ModTime.IsZero() {
		t.Fatalf("Extract() returned file with zero modtime")
	}
	assert.Equalf(t, content, files[0].Data.String(), "Extract() returned file with content %s, expected %s", files[0].Data.String(), content)
	// assert extracted filepaths are sanitized
	assert.Equalf(t, "hack.txt", files[1].Name, "filepath is not sanitized: %s", files[1].Name)
	// assert there are 2 files in the tarball
	assert.Lenf(t, files, 2, "Extract() returned %d files, expected 2", len(files))

}

func TestTarball_Create(t *testing.T) {
	t.Parallel()

	files := []*File{
		{Name: "testfile.txt", Mode: 0644, Size: 9, ModTime: time.Now(), Data: bytes.NewBufferString("testdata\n")},
	}

	var buf bytes.Buffer
	tarball := NewTarballService()
	if err := tarball.Create(&buf, files); err != nil {
		t.Fatalf("error creating tarball: %v", err)
	}

	// read the tarball back
	gzipReader, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("error creating gzip reader: %v", err)
	}
	tr := tar.NewReader(gzipReader)
	header, err := tr.Next()
	if err != nil {
		t.Fatalf("error reading tarball: %v", err)
	}
	assert.Equalf(t, "testfile.txt", header.Name, "unexpected file in tarball: %s", header.Name)
	assert.Equalf(t, int64(0644), header.Mode, "unexpected file mode in tarball: %o", header.Mode)
	assert.Equalf(t, int64(9), header.Size, "unexpected file size in tarball: %d", header.Size)

	//buf.Reset()
	var decoded bytes.Buffer
	if _, err = io.Copy(&decoded, tr); err != nil {
		t.Fatalf("error copying tarball contents: %v", err)
	}
	assert.Equalf(t, "testdata\n", decoded.String(), "unexpected file contents in tarball: %s", decoded.String())
}
