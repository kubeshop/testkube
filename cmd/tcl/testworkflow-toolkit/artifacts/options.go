// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"net/http"

	"github.com/minio/minio-go/v7"
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

var DirectDetectMimetype = WithMinioOptionsEnhancer(func(options *minio.PutObjectOptions, path string, size int64) {
	if options.ContentType == "" {
		options.ContentType = DetectMimetype(path)
	}
})
