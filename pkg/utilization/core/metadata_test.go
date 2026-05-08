package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestMetadataString(t *testing.T) {
	tests := []struct {
		name string
		meta Metadata
		want string
	}{
		{
			name: "all fields set",
			meta: Metadata{
				Workflow:  "wf",
				Step:      Step{Ref: "s1", Name: "Some step"},
				Execution: "ex1",
				Lines:     10,
				Format:    FormatInflux,
				ContainerResources: ContainerResources{
					Requests: ResourceList{
						CPU:    "100m",
						Memory: "256Mi",
					},
					Limits: ResourceList{
						CPU:    "200m",
						Memory: "512Mi",
					},
				},
			},
			want: "META workflow=wf step.ref=s1 step.name=\"Some step\" execution=ex1 lines=10 format=influx resources.requests.cpu=100m resources.requests.memory=256Mi resources.limits.cpu=200m resources.limits.memory=512Mi",
		},
		{
			name: "some fields empty",
			meta: Metadata{
				Workflow: "wf2",
				// Step is empty
				// Execution is empty
				Lines:  0, // zero value (should be omitted)
				Format: FormatCSV,
				ContainerResources: ContainerResources{
					// Requests are both empty
					Limits: ResourceList{
						CPU: "500m",
					},
				},
			},
			want: "META workflow=wf2 format=csv resources.limits.cpu=500m",
		},
		{
			name: "completely empty",
			meta: Metadata{},
			want: "META",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, tt.meta.String())
		})
	}
}

func TestParseMetadataFromFilename(t *testing.T) {
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
				Step:      Step{Ref: "step2"},
				Execution: "0002",
				Format:    FormatInflux,
			},
			wantErr:    false,
			errMessage: "",
		},
		{
			name:     "valid INFLUX file with resource",
			filename: "myWorkflow_step2_0002.influx",
			wantMeta: &Metadata{
				Workflow:  "myWorkflow",
				Step:      Step{Ref: "step2"},
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
			filename:   "workflow_step_execution_as.json",
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

func TestParseMetadataFromHeader(t *testing.T) {
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
				Step:      Step{Ref: "step", Name: "Some step"},
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
	tests := []struct {
		name       string
		input      string
		wantMeta   *Metadata
		wantErr    bool
		errContain string
	}{
		{
			name:  "Valid - influx",
			input: `META lines=10 format=influx step.ref=step1 step.name="Step 1" workflow=wf execution=ex1 resources.requests.cpu=100m resources.requests.memory=256 resources.limits.cpu=200m resources.limits.memory=512`,
			wantMeta: &Metadata{
				Workflow:  "wf",
				Step:      Step{Ref: "step1", Name: "Step 1"},
				Execution: "ex1",
				Lines:     10,
				Format:    FormatInflux,
				ContainerResources: ContainerResources{
					Requests: ResourceList{
						CPU:    "100m",
						Memory: "256",
					},
					Limits: ResourceList{
						CPU:    "200m",
						Memory: "512",
					},
				},
			},
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
