package export

import "testing"

func TestParseOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		line   string
		want   string
		wantOK bool
	}{
		{
			name:   "json structured log",
			line:   `{"level":"info","msg":"usage export complete","outputPath":"/output/plan-usage-acme-20260101T120000Z.zip"}`,
			want:   "/output/plan-usage-acme-20260101T120000Z.zip",
			wantOK: true,
		},
		{
			name:   "logfmt style",
			line:   `usage export complete outputPath=/output/plan-usage.zip sizeBytes=1234`,
			want:   "/output/plan-usage.zip",
			wantOK: true,
		},
		{
			name:   "download hint line",
			line:   "Download: kubectl -n testkube cp usage-export-pod:/output/plan-usage.zip ./plan-usage.zip",
			want:   "/output/plan-usage.zip",
			wantOK: true,
		},
		{
			name:   "unrelated line",
			line:   "connected to postgres",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseOutputPath(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("ParseOutputPath() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("ParseOutputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultLocalOutput(t *testing.T) {
	t.Parallel()

	if got := defaultLocalOutput("/output/plan-usage-acme.zip"); got != "plan-usage-acme.zip" {
		t.Fatalf("defaultLocalOutput() = %q", got)
	}
}
