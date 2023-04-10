package artifact

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdArtifactsListFiles       executor.Command = "artifacts.listFiles"
	CmdArtifactsDownloadFile    executor.Command = "artifacts.downloadFile"
	CmdArtifactsDownloadArchive executor.Command = "artifacts.downloadArchive"
)
