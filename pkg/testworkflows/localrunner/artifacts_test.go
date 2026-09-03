package localrunner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

type artifactArchiveEntry struct {
	name     string
	typeflag byte
	contents string
}

func TestPrepareArtifactPlanDoesNotCreateDestinationAndRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "new", "artifacts")
	plan, err := PrepareArtifactPlan(destination, 1024)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, destination, plan.Destination)
	_, statErr := os.Stat(filepath.Join(root, "new"))
	assert.True(t, os.IsNotExist(statErr), "Prepare must not create a host output directory")

	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(root, link))
	_, err = PrepareArtifactPlan(filepath.Join(link, "artifacts"), 1024)
	require.Error(t, err)
	assert.True(t, IsUsageError(err))
	assert.Contains(t, err.Error(), "symbolic link")
}

func TestPrepareArtifactPlanRejectsArchiveLimitOverflowBeforeClusterMutation(t *testing.T) {
	_, err := PrepareArtifactPlan(t.TempDir(), math.MaxInt64)
	require.Error(t, err)
	assert.True(t, IsUsageError(err))
	assert.Contains(t, err.Error(), "max artifact bytes")
}

func TestExtractLocalArtifactArchivePublishesOnlyRegularFiles(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "published")
	archive := artifactArchive(t, []artifactArchiveEntry{
		{name: "./", typeflag: tar.TypeDir},
		{name: "./steps/", typeflag: tar.TypeDir},
		{name: "./steps/0/", typeflag: tar.TypeDir},
		{name: "steps/0/report.txt", typeflag: tar.TypeReg, contents: "artifact report"},
	})
	summary, err := extractLocalArtifactArchive(context.Background(), bytes.NewReader(archive), destination, 1024)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Files)
	assert.EqualValues(t, len("artifact report"), summary.Bytes)
	contents, err := os.ReadFile(filepath.Join(destination, "steps", "0", "report.txt"))
	require.NoError(t, err)
	assert.Equal(t, "artifact report", string(contents))
	info, err := os.Stat(filepath.Join(destination, "steps", "0", "report.txt"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSafeArtifactArchiveNamePermitsOnlyCanonicalDirectorySuffix(t *testing.T) {
	name, root, err := safeArtifactArchiveName("./steps/", true)
	require.NoError(t, err)
	assert.Equal(t, "steps", name)
	assert.False(t, root)

	_, _, err = safeArtifactArchiveName("./steps//", true)
	require.Error(t, err)
	_, _, err = safeArtifactArchiveName("./steps/", false)
	require.Error(t, err)
}

func TestExtractLocalArtifactArchiveRejectsUnsafeEntries(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry artifactArchiveEntry
	}{
		{name: "parent traversal", entry: artifactArchiveEntry{name: "../outside", typeflag: tar.TypeReg, contents: "no"}},
		{name: "symlink", entry: artifactArchiveEntry{name: "steps/link", typeflag: tar.TypeSymlink}},
		{name: "backslash", entry: artifactArchiveEntry{name: `steps\\report.txt`, typeflag: tar.TypeReg, contents: "no"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			destination := filepath.Join(root, "artifacts")
			_, err := extractLocalArtifactArchive(context.Background(), bytes.NewReader(artifactArchive(t, []artifactArchiveEntry{test.entry})), destination, 1024)
			require.Error(t, err)
			_, statErr := os.Stat(filepath.Join(root, "outside"))
			assert.True(t, os.IsNotExist(statErr))
		})
	}
}

func TestExtractLocalArtifactArchiveHonorsUncompressedByteLimit(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifacts")
	_, err := extractLocalArtifactArchive(context.Background(), bytes.NewReader(artifactArchive(t, []artifactArchiveEntry{{name: "steps/0/large.txt", typeflag: tar.TypeReg, contents: "too-large"}})), destination, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max artifact bytes")
	_, statErr := os.Stat(filepath.Join(destination, "steps", "0", "large.txt"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestArtifactRelayResourcesUseExactLabelsAndSecretToken(t *testing.T) {
	labels, err := Labels("local-artifact-test", "artifact-relay")
	require.NoError(t, err)
	secret := artifactRelaySecret("relay-token", DefaultNamespace, labels, "secret-token")
	pod := artifactRelayPod("relay", DefaultNamespace, labels, secret.Name, 1024)
	assert.Equal(t, labels, secret.Labels)
	assert.Equal(t, "secret-token", secret.StringData["token"])
	assert.Equal(t, labels, pod.Labels)
	assert.Nil(t, pod.Spec.ActiveDeadlineSeconds)
	require.NotNil(t, pod.Spec.AutomountServiceAccountToken)
	assert.False(t, *pod.Spec.AutomountServiceAccountToken)
	require.Len(t, pod.Spec.Containers, 1)
	env := pod.Spec.Containers[0].Env[0]
	require.NotNil(t, env.ValueFrom)
	require.NotNil(t, env.ValueFrom.SecretKeyRef)
	assert.Equal(t, secret.Name, env.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", env.ValueFrom.SecretKeyRef.Key)
	require.NotNil(t, pod.Spec.Containers[0].SecurityContext)
	assert.Equal(t, []corev1.Capability{"ALL"}, pod.Spec.Containers[0].SecurityContext.Capabilities.Drop)
}

func TestArtifactRelayURLUsesTheExactReadyPodAddress(t *testing.T) {
	assert.Equal(t, "http://10.244.1.9:8080/upload", artifactRelayURL("10.244.1.9"))
	assert.Equal(t, "http://[fd00::9]:8080/upload", artifactRelayURL("fd00::9"))
}

func artifactArchive(t *testing.T, entries []artifactArchiveEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Typeflag: entry.typeflag, Mode: 0o777, Size: int64(len(entry.contents))}
		if entry.typeflag == tar.TypeDir {
			header.Size = 0
		}
		require.NoError(t, tarWriter.WriteHeader(header))
		if entry.contents != "" {
			_, err := tarWriter.Write([]byte(entry.contents))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return output.Bytes()
}
