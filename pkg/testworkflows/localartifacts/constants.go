// Package localartifacts defines the private protocol used by testkube local
// to move TestWorkflow artifacts from a workflow Pod to its run-owned relay.
//
// The protocol intentionally keeps its bearer token in a Kubernetes Secret
// environment variable. It must not be copied into TK_CFG, Job annotations, or
// command-line arguments.
package localartifacts

import (
	"fmt"
	"path"
	"strings"
)

const (
	// MaxArtifactFiles and the portable path bounds keep one relay archive
	// comfortably below the local runner's hardened extraction entry limit.
	// Artifact export is a developer-loop facility, not an unbounded file sync.
	MaxArtifactFiles          = 1_000
	MaxRelativePathLength     = 4_096
	MaxRelativePathComponents = 8

	// TokenEnvName is the environment variable containing the relay bearer
	// token. Both the artifact toolkit stage and the relay Pod receive it from
	// the same Secret key.
	TokenEnvName = "TK_LOCAL_ARTIFACT_TOKEN"

	// TokenSecretKey is the key expected in the run-owned relay Secret.
	TokenSecretKey = "token"

	// TokenHeader authenticates upload requests to the local artifact relay.
	TokenHeader = "X-Testkube-Local-Artifact-Token"

	// PathHeader carries a slash-separated, relative artifact destination.
	// Keeping it out of the URL avoids path-normalization ambiguities between
	// the uploader and net/http.
	PathHeader = "X-Testkube-Local-Artifact-Path"
)

// ValidateRelativePath accepts only a non-empty, portable relative path. The
// relay protocol uses POSIX separators regardless of the machine that starts
// testkube local, so reject backslashes instead of interpreting them
// differently at either end of the connection.
func ValidateRelativePath(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("path must not contain NUL")
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf("path must use forward slashes")
	}
	if strings.Contains(value, ":") {
		return "", fmt.Errorf("path must not contain a volume separator")
	}
	if len(value) > MaxRelativePathLength {
		return "", fmt.Errorf("path exceeds %d bytes", MaxRelativePathLength)
	}
	if strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("path must be relative")
	}

	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || path.IsAbs(cleaned) {
		return "", fmt.Errorf("path must stay below the artifact root")
	}
	if len(strings.Split(cleaned, "/")) > MaxRelativePathComponents {
		return "", fmt.Errorf("path exceeds %d components", MaxRelativePathComponents)
	}
	return cleaned, nil
}

// ValidateStepRef accepts an artifact-stage reference for use as a single
// directory below steps/. A reference may not introduce another hierarchy.
func ValidateStepRef(value string) (string, error) {
	cleaned, err := ValidateRelativePath(value)
	if err != nil {
		return "", err
	}
	if strings.Contains(cleaned, "/") {
		return "", fmt.Errorf("step reference must be a single path segment")
	}
	return cleaned, nil
}
