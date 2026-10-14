package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/common"
)

func TestReadStepError(t *testing.T) {
	secret := "ghp_0123456789abcdefghijklmnopqrstuvwxyz"

	tests := []struct {
		name            string
		content         *string
		sensitiveValues []string
		want            string
	}{
		{
			name: "returns an empty message when the toolkit wrote no file",
			want: "",
		},
		{
			name:    "keeps only the first line",
			content: common.Ptr("2 of 5 executions failed: a-1 (failed), b-2 (aborted)\nhint: second line\n"),
			want:    "2 of 5 executions failed: a-1 (failed), b-2 (aborted)",
		},
		{
			name:    "stops at a carriage return",
			content: common.Ptr("finishing upload: connection refused\rnext"),
			want:    "finishing upload: connection refused",
		},
		{
			name:    "removes the ANSI escape codes",
			content: common.Ptr("\x1b[91mbroken-service\x1b[0m: compute matrix and sharding"),
			want:    "broken-service: compute matrix and sharding",
		},
		{
			name:    "keeps at most 1 KiB",
			content: common.Ptr(strings.Repeat("a", 2000)),
			want:    strings.Repeat("a", 1024),
		},
		{
			name:            "masks a sensitive value that crosses the size limit",
			content:         common.Ptr(strings.Repeat("a", 1000) + secret),
			sensitiveValues: []string{secret},
			want:            strings.Repeat("a", 1000) + "*****",
		},
		{
			name:            "masks a sensitive value that crosses the end of the first line",
			content:         common.Ptr("token ghp_0123\n456789"),
			sensitiveValues: []string{"ghp_0123\n456789"},
			want:            "token *****",
		},
		{
			name:    "removes a multibyte character that the limit cuts",
			content: common.Ptr(strings.Repeat("a", 1023) + "é"),
			want:    strings.Repeat("a", 1023),
		},
		{
			name:    "removes the escape codes before it limits the size",
			content: common.Ptr(strings.Repeat("a", 1020) + "\x1b[91m" + "bbbbbb"),
			want:    strings.Repeat("a", 1020) + "bbbb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "error")
			if tt.content != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.content), 0666))
			}

			assert.Equal(t, tt.want, readStepError(path, tt.sensitiveValues))
		})
	}
}
