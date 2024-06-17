package artifacts

import (
	"path/filepath"

	"github.com/h2non/filetype"
)

func DetectMimetype(filePath string) string {
	ext := filepath.Ext(filePath)

	// Remove the dot from the file extension
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	t := filetype.GetType(ext)
	if t == filetype.Unknown {
		return ""
	}
	return t.MIME.Value
}
