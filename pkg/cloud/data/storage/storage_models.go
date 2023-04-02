package storage

import (
	"io"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ListFilesRequest struct {
	BucketFolder string
}

type ListFilesResponse struct {
	Artifacts []testkube.Artifact
}

type DeleteFileRequest struct {
	BucketFolder string
	File         string
}

type DeleteFileResponse struct{}

type DownloadFileRequest struct {
	BucketFolder string
	File         string
}

type DownloadFileResponse struct {
	URL string
}

type SaveFileRequest struct {
	BucketFolder string
	FilePath     string
	Reader       io.Reader
	ObjectSize   int64
}

type SaveFileResponse struct{}

type PlaceFilesRequest struct {
	BucketFolders []string
	Prefix        string
}

type PlaceFilesResponse struct{}
