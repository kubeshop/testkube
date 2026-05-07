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
	// plain text CA certificate to verify repository TLS connection
	CaCert string `json:"caCert,omitempty"`
	// external CA certificate to verify repository TLS connection
	CaCertFrom *corev1.EnvVarSource `json:"caCertFrom,omitempty" expr:"force"`
	// plain text client certificate to verify client (testkube) TLS connection
	ClientCert string `json:"clientCert,omitempty"`
	// external client certificate to verify client (testkube) TLS connection
	ClientCertFrom *corev1.EnvVarSource `json:"clientCertFrom,omitempty" expr:"force"`
	// plain text client key to verify client (testkube) TLS connection
	ClientKey string `json:"clientKey,omitempty"`
	// external client key to verify client (testkube) TLS connection
	ClientKeyFrom *corev1.EnvVarSource `json:"clientKeyFrom,omitempty" expr:"force"`
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

type Content struct {
	// git repository details
	Git *ContentGit `json:"git,omitempty" expr:"include"`
	// files to load
	Files []ContentFile `json:"files,omitempty" expr:"include"`
	// tarballs to unpack
	Tarball []ContentTarball `json:"tarball,omitempty" expr:"include"`
}
