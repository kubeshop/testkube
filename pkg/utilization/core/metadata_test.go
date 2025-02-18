package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestWriteMetadataToFile(t *testing.T) {
	t.Parallel()

	tmpFile, err := os.CreateTemp("", "test-write-metadata-*.influx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name()) // clean up

	_, err = tmpFile.WriteString(`cpu,host=server01 usage_idle=99
mem,host=server01 usage=2048
`)
	require.NoError(t, err)

	metadata := &Metadata{
		Lines:  2,
		Format: FormatInflux,
	}

	err = WriteMetadataToFile(tmpFile, metadata)
	require.NoError(t, err)

	expectedContent := `cpu,host=server01 usage_idle=99
mem,host=server01 usage=2048
#META lines=2 format=influx
`

	actualContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, expectedContent, string(actualContent))
}

func TestParseMetadataFromFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		fileName        string
		expectErr       bool
		expectErrSubstr string
		expectMetadata  *Metadata
	}{
		{
			name:     "Valid metadata file",
			fileName: "metrics_valid_metadata.influx",
			expectMetadata: &Metadata{
				Lines:  50,
				Format: FormatInflux,
			},
		},
		{
			name:           "File without metadata",
			fileName:       "metrics_no_metadata.influx",
			expectMetadata: &Metadata{Lines: 0, Format: FormatInflux},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.fileName)
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("Failed to open file %s: %v", path, err)
			}
			t.Cleanup(func() {
				f.Close()
			})

			meta, err := parseMetadataFromFile(f)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.expectErrSubstr != "" {
					assert.Contains(t, err.Error(), tc.expectErrSubstr)
				}
				assert.Nil(t, meta)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, meta)
				assert.Equal(t, tc.expectMetadata, meta)
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantMeta   *Metadata
		wantErr    bool
		errContain string
	}{
		{
			name:     "Valid - influx",
			input:    "#META lines=10 format=influx",
			wantMeta: &Metadata{Lines: 10, Format: FormatInflux},
		},
		{
			name:     "Valid - csv",
			input:    "#META lines=42 format=csv",
			wantMeta: &Metadata{Lines: 42, Format: FormatCSV},
		},
		{
			name:     "Valid - json",
			input:    "#META lines=7 format=json",
			wantMeta: &Metadata{Lines: 7, Format: FormatJSON},
		},
		{
			name:       "Invalid - missing prefix",
			input:      "lines=10 format=influx",
			wantErr:    true,
			errContain: "meta line must start with",
		},
		{
			name:       "Invalid - only one token",
			input:      "#META lines=10",
			wantErr:    true,
			errContain: "format", // we expect an error about missing format key
		},
		{
			name:       "Invalid - cannot parse lines as int",
			input:      "#META lines=abc format=csv",
			wantErr:    true,
			errContain: "failed to parse 'lines' as int",
		},
		{
			name:       "Invalid - unrecognized key",
			input:      "#META lines=5 something=wrong format=influx",
			wantErr:    true,
			errContain: "unrecognized metadata key",
		},
		{
			name:       "Invalid - unsupported format",
			input:      "#META lines=5 format=toml",
			wantErr:    true,
			errContain: "unsupported format",
		},
		{
			name:     "Valid - no tokens after prefix",
			input:    "#META ",
			wantMeta: &Metadata{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotMeta, err := parseMetadata(tc.input)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errContain != "" {
					assert.Contains(t, err.Error(), tc.errContain)
				}
				assert.Nil(t, gotMeta)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gotMeta)
				assert.Equal(t, tc.wantMeta, gotMeta)
			}
		})
	}
}
