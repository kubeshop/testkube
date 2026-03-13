package v1

import (
	corev1 "k8s.io/api/core/v1"

	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
)

type ContentGit struct {
	// uri for the Git repository
	Uri string `json:"uri,omitempty" expr:"template"`
	// branch, commit or a tag name to fetch
	Revision string `json:"revision,omitempty" expr:"template"`
	// plain text username to fetch with
	Username string `json:"username,omitempty" expr:"template"`
	// external username to fetch with
	UsernameFrom *corev1.EnvVarSource `json:"usernameFrom,omitempty" expr:"force"`
	// plain text token to fetch with
	Token string `json:"token,omitempty" expr:"template"`
	// external token to fetch with
	TokenFrom *corev1.EnvVarSource `json:"tokenFrom,omitempty" expr:"force"`
	// plain text SSH private key to fetch with
	SshKey string `json:"sshKey,omitempty" expr:"template"`
	// external SSH private key to fetch with
	SshKeyFrom *corev1.EnvVarSource `json:"sshKeyFrom,omitempty" expr:"force"`
	// authorization type for the credentials
	AuthType testsv3.GitAuthType `json:"authType,omitempty" expr:"template"`
	// where to mount the fetched repository contents (defaults to "repo" directory in the data volume)
	MountPath string `json:"mountPath,omitempty" expr:"template"`
	// enable cone mode for sparse checkout with paths
	Cone bool `json:"cone,omitempty" expr:"ignore"`
	// paths to fetch for the sparse checkout
	Paths []string `json:"paths,omitempty" expr:"template"`
}

type ContentFile struct {
	// path where the file should be accessible at
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path" expr:"template"`
	// plain-text content to put inside
	Content string `json:"content,omitempty" expr:"template"`
	// external source to use
	ContentFrom *corev1.EnvVarSource `json:"contentFrom,omitempty" expr:"force"`
	// mode to use for the file
	Mode *int32 `json:"mode,omitempty"`
}

type ContentTarball struct {
	// url for the tarball to extract
	Url string `json:"url" expr:"template"`
	// path where the tarball should be extracted
	Path string `json:"path" expr:"template"`
	// should it mount a new volume there
	Mount *bool `json:"mount,omitempty" expr:"ignore"`
}

type ContentMinio struct {
	// endpoint for the MinIO/S3 storage
	Endpoint string `json:"endpoint,omitempty" expr:"template"`
	// bucket name to fetch from
	Bucket string `json:"bucket,omitempty" expr:"template"`
	// path within the bucket to fetch
	Path string `json:"path,omitempty" expr:"template"`
	// region for the storage
	Region string `json:"region,omitempty" expr:"template"`
	// plain text access key to fetch with
	AccessKey string `json:"accessKey,omitempty" expr:"template"`
	// external access key to fetch with
	AccessKeyFrom *corev1.EnvVarSource `json:"accessKeyFrom,omitempty" expr:"force"`
	// plain text secret key to fetch with
	SecretKey string `json:"secretKey,omitempty" expr:"template"`
	// external secret key to fetch with
	SecretKeyFrom *corev1.EnvVarSource `json:"secretKeyFrom,omitempty" expr:"force"`
	// where to mount the fetched bucket contents (defaults to "minio" directory in the data volume)
	MountPath string `json:"mountPath,omitempty" expr:"template"`
}

type Content struct {
	// git repository details
	Git *ContentGit `json:"git,omitempty" expr:"include"`
	// files to load
	Files []ContentFile `json:"files,omitempty" expr:"include"`
	// tarballs to unpack
	Tarball []ContentTarball `json:"tarball,omitempty" expr:"include"`
	// MinIO/S3 storage details
	Minio *ContentMinio `json:"minio,omitempty" expr:"include"`
}
