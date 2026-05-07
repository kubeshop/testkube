package commands

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// setupCertAuth sets up TLS certificate-based authentication for git by writing
// cert/key/CA data to temporary files and returning the git config args needed
// to use them. The caller must invoke all returned cleanup functions when done.
func setupCertAuth(opts *CloneOptions) (args []string, cleanups []func(), err error) {
	cleanups = make([]func(), 0, 3)

	if opts.CaCert != "" {
		caPath, cleanup, writeErr := writeTempCertFile(opts.CaCert, "git-ca-cert-*")
		if writeErr != nil {
			RunCleanupFuncs(cleanups)
			return nil, nil, fmt.Errorf("writing CA certificate: %w", writeErr)
		}
		cleanups = append(cleanups, cleanup)
		args = append(args, "-c", fmt.Sprintf("http.sslCAInfo=%s", caPath))
	}

	if opts.ClientCert != "" {
		certPath, cleanup, writeErr := writeTempCertFile(opts.ClientCert, "git-client-cert-*")
		if writeErr != nil {
			RunCleanupFuncs(cleanups)
			return nil, nil, fmt.Errorf("writing client certificate: %w", writeErr)
		}
		cleanups = append(cleanups, cleanup)
		args = append(args, "-c", fmt.Sprintf("http.sslCert=%s", certPath))
	}

	if opts.ClientKey != "" {
		keyPath, cleanup, writeErr := writeTempCertFile(opts.ClientKey, "git-client-key-*")
		if writeErr != nil {
			RunCleanupFuncs(cleanups)
			return nil, nil, fmt.Errorf("writing client key: %w", writeErr)
		}
		cleanups = append(cleanups, cleanup)
		args = append(args, "-c", fmt.Sprintf("http.sslKey=%s", keyPath))
	}

	return args, cleanups, nil
}

// writeTempCertFile writes content to a temporary file with restricted permissions
// and returns the file path and a cleanup function to remove it.
func writeTempCertFile(content, pattern string) (string, func(), error) {
	tmpFile, err := os.CreateTemp(constants.DefaultTmpDirPath, pattern)
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}
	path := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(path, 0400); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("setting permissions on temp file: %w", err)
	}

	cleanup := func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

// RunCleanupFuncs executes all provided cleanup functions.
func RunCleanupFuncs(cleanups []func()) {
	for _, fn := range cleanups {
		fn()
	}
}
