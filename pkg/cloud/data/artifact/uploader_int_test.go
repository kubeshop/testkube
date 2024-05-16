package artifact

import "testing"

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "PDF file",
			filePath: "document.pdf",
			want:     "application/pdf",
		},
		{
			name:     "MP4 file",
			filePath: "video.mp4",
			want:     "video/mp4",
		},
		{
			name:     "XML file",
			filePath: "data.xml",
			want:     "text/xml",
		},
		{
			name:     "jtl file",
			filePath: "file.jtl",
			want:     "text/plain",
		},
		{
			name:     "log file",
			filePath: "file.log",
			want:     "text/plain",
		},
		{
			name:     "tar ball",
			filePath: "artifacts.tar.gz",
			want:     "application/gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getContentType(tt.filePath); got != tt.want {
				t.Errorf("GetContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}
