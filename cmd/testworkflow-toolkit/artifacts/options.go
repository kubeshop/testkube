package artifacts

import (
	"net/http"
)

var CloudDetectMimetype = WithRequestEnhancerCloud(func(req *http.Request, path string, size int64) {
	if req.Header.Get("Content-Type") == "" {
		contentType := DetectMimetype(path)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if contentType == "application/gzip" && req.Header.Get("Content-Encoding") == "" {
			req.Header.Set("Content-Encoding", "gzip")
		}
	}
})
