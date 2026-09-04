package common

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/require"
)

// buildDependencyTreeArchive mimics the shape a dependency cache actually has: many
// small files spread thinly over many directories, with a few large ones among them.
// The shape is what costs, not the volume - the per-entry work is what a restore spends
// its time on.
func buildDependencyTreeArchive(t testing.TB, dirs, filesPerDir int) []byte {
	t.Helper()

	rnd := rand.New(rand.NewSource(1))
	small := make([]byte, 3<<10)
	rnd.Read(small)
	large := make([]byte, 4<<20)
	rnd.Read(large)

	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	// Deep-ish names, like github.com/org/repo@v1.2.3/internal/pkg/file.go.
	for d := 0; d < dirs; d++ {
		dir := path.Join(
			"deps", fmt.Sprintf("host%d", d%4), fmt.Sprintf("org%d", d%37),
			fmt.Sprintf("repo%d@v1.%d.0", d, d%9), "internal", fmt.Sprintf("pkg%d", d%11),
		)
		for f := 0; f < filesPerDir; f++ {
			body := small
			if d%64 == 0 && f == 0 {
				body = large
			}
			name := path.Join(dir, fmt.Sprintf("file%d.go", f))
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     name,
				Mode:     0o644,
				Size:     int64(len(body)),
			}))
			_, err := tw.Write(body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func BenchmarkUnpackDependencyTree(b *testing.B) {
	archive := buildDependencyTreeArchive(b, 300, 8)
	b.Logf("archive: %d entries, %.1f MiB compressed", 300*8, float64(len(archive))/(1<<20))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		destination := b.TempDir()
		b.StartTimer()
		if err := UnpackTarball(destination, bytes.NewReader(archive)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPackDependencyTree measures the other half. The shape is the same, because
// both halves are bound by per-entry work rather than by volume.
func BenchmarkPackDependencyTree(b *testing.B) {
	source := b.TempDir()
	if filepath.VolumeName(source) != "" {
		b.Skip("packing walks from \"/\"; a Windows drive path cannot be reached from there")
	}
	const dirs, filesPerDir = 300, 8
	body := make([]byte, 3<<10)
	for d := 0; d < dirs; d++ {
		dir := filepath.Join(source, fmt.Sprintf("host%d", d%4), fmt.Sprintf("repo%d@v1.0.0", d), "internal")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		for f := 0; f < filesPerDir; f++ {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", f)), body, 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	root := filepath.ToSlash(source)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entries, err := WriteTarballFrom(io.Discard, "/", []string{root, path.Join(root, "**")}, []string{root})
		if err != nil {
			b.Fatal(err)
		}
		if entries != dirs*filesPerDir {
			b.Fatalf("packed %d entries, want %d", entries, dirs*filesPerDir)
		}
	}
}
