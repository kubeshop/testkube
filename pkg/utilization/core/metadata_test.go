package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseMetadataFromFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filename   string
		wantMeta   *Metadata
		wantErr    bool
		errMessage string
	}{
		{
			name:     "valid INFLUX file",
			filename: "myWorkflow_step2_0002.influx",
			wantMeta: &Metadata{
				Workflow:  "myWorkflow",
				Step:      "step2",
				Execution: "0002",
				Format:    FormatInflux,
			},
			wantErr:    false,
			errMessage: "",
		},
		{
			name:       "invalid extension",
			filename:   "someWorkflow_someStep_someExecution.txt",
			wantMeta:   nil,
			wantErr:    true,
			errMessage: "unsupported metrics file extension",
		},
		{
			name:       "invalid format - fewer underscore segments",
			filename:   "someWorkflow_onlyOneSegment.json",
			wantMeta:   nil,
			wantErr:    true,
			errMessage: "invalid filename format: expected <workflow>_<step>_<execution>.<format>",
		},
		{
			name:       "invalid format - more underscore segments",
			filename:   "workflow_step_execution_extra.json",
			wantMeta:   nil,
			wantErr:    true,
			errMessage: "invalid filename format: expected <workflow>_<step>_<execution>.<format>",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			gotMeta, err := parseMetadataFromFilename(tt.filename)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, gotMeta, "Metadata should not be nil when no error")
				assert.Equal(t, tt.wantMeta, gotMeta)
			}
		})
	}
}

func TestReadHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fileName       string
		expectMetadata *Metadata
		expectErr      bool
		errContains    string
	}{
		{
			name:     "metrics file with valid metadata",
			fileName: "metrics_valid_metadata.influx",
			expectMetadata: &Metadata{
				Lines:     50,
				Format:    FormatInflux,
				Workflow:  "workflow",
				Step:      "step",
				Execution: "execution",
			},
		},
		{
			name:        "metrics file with no metadata",
			fileName:    "metrics_no_metadata.influx",
			expectErr:   true,
			errContains: "invalid header control byte 'c': no metadata found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.fileName)
			f, err := os.Open(path)
			t.Cleanup(func() {
				_ = f.Close()
			})
			require.NoError(t, err)

			buf := make([]byte, headerLength)
			_, err = f.Read(buf)
			require.NoError(t, err)

			meta, err := parseMetadataFromHeader(buf)
			if tc.expectErr {
				assert.ErrorContains(t, err, tc.errContains)
			} else {
				require.NoError(t, err)
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
			input:    "META lines=10 format=influx",
			wantMeta: &Metadata{Lines: 10, Format: FormatInflux},
		},
		{
			name:     "Valid - csv",
			input:    "META lines=42 format=csv",
			wantMeta: &Metadata{Lines: 42, Format: FormatCSV},
		},
		{
			name:     "Valid - json",
			input:    "META lines=7 format=json",
			wantMeta: &Metadata{Lines: 7, Format: FormatJSON},
		},
		{
			name:       "Invalid - missing prefix",
			input:      "lines=10 format=influx",
			wantErr:    true,
			errContain: "meta line must start with",
		},
		{
			name:       "Invalid - cannot parse lines as int",
			input:      "META lines=abc format=csv",
			wantErr:    true,
			errContain: "failed to parse 'lines' as int",
		},
		{
			name:       "Invalid - unrecognized key",
			input:      "META lines=5 something=wrong format=influx",
			wantErr:    true,
			errContain: "unrecognized metadata key",
		},
		{
			name:       "Invalid - unsupported format",
			input:      "META lines=5 format=toml",
			wantErr:    true,
			errContain: "unsupported metrics format",
		},
		{
			name:     "Valid - no tokens after prefix",
			input:    "META ",
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
