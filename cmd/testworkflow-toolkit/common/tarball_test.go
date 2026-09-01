package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildMixedTarball(t *testing.T, regContent string) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}))

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/file.txt",
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len(regContent)),
	}))
	_, err := tw.Write([]byte(regContent))
	require.NoError(t, err)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "sub/link.txt",
		Typeflag: tar.TypeSymlink,
		Linkname: "file.txt",
		Mode:     0o777,
	}))

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf
}

func TestUnpackTarball_ExtractIsIdempotentOnRetry(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "hello")))
	assert.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "hello")))

	data, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	target, err := os.Readlink(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "file.txt", target)
}

func TestUnpackTarball_ExtractOverwritesOnRetry(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "first")))

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "sub/", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "sub/other.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 5,
	}))
	_, err := tw.Write([]byte("other"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "sub/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 6}))
	_, err = tw.Write([]byte("second"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "sub/link.txt", Typeflag: tar.TypeSymlink, Linkname: "other.txt", Mode: 0o777,
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	require.NoError(t, UnpackTarball(dir, buf))

	got, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "second", string(got))

	target, err := os.Readlink(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "other.txt", target)
}

func TestUnpackTarball_FreshExtract(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, UnpackTarball(dir, buildMixedTarball(t, "payload")))

	data, err := os.ReadFile(filepath.Join(dir, "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))

	_, err = os.Lstat(filepath.Join(dir, "sub", "link.txt"))
	require.NoError(t, err)
}
