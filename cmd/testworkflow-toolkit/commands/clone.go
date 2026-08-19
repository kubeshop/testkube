package commands

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	// CloneRetryOnFailureMaxAttempts defines default retry attempts for git operations
	CloneRetryOnFailureMaxAttempts = 5
	// CloneRetryOnFailureBaseDelay defines default base delay between retries
	CloneRetryOnFailureBaseDelay = 100 * time.Millisecond
	// CloneRetryCountCap is the maximum allowed retry attempts (inclusive)
	CloneRetryCountCap = 20
	// CloneRetryDelayCap is the maximum allowed base delay between retries
	CloneRetryDelayCap = 30 * time.Second
)

var (
	// protocolRe matches URI schemes like http://, https://, ssh://
	protocolRe = regexp.MustCompile(`^[^:]+://`)
)

// CloneOptions encapsulates all options for the clone command
type CloneOptions struct {
	RawPaths   []string
	Username   string
	Token      string
	SSHKey     string
	AuthType   string
	Revision   string
	Cone       bool
	Verbosity  string
	RetryCount int
	RetryDelay time.Duration
}

// NewCloneCmd creates a new clone command
func NewCloneCmd() *cobra.Command {
	opts := &CloneOptions{}

	cmd := &cobra.Command{
		Use:   "clone <uri> <outputPath>",
		Short: "Clone the Git repository",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunClone(cmd.Context(), args[0], args[1], opts); err != nil {
				ui.Fail(err)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&opts.RawPaths, "paths", "p", nil, "paths for sparse checkout")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "git username for authentication")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "git token for authentication")
	cmd.Flags().StringVarP(&opts.SSHKey, "sshKey", "s", "", "SSH private key for authentication")
	cmd.Flags().StringVarP(&opts.AuthType, "authType", "a", "basic", "authentication type (allowed: basic, header, github)")
	cmd.Flags().StringVarP(&opts.Revision, "revision", "r", "", "commit hash, branch name or tag")
	cmd.Flags().BoolVar(&opts.Cone, "cone", false, "enable cone mode for sparse checkout")
	cmd.Flags().StringVar(&opts.Verbosity, "verbosity", "verbose", "logging level (allowed: quiet, normal, verbose)")
	cmd.Flags().IntVar(&opts.RetryCount, "retry-count", CloneRetryOnFailureMaxAttempts, fmt.Sprintf("max attempts for transient git failures (capped at %d)", CloneRetryCountCap))
	cmd.Flags().DurationVar(&opts.RetryDelay, "retry-delay", CloneRetryOnFailureBaseDelay, fmt.Sprintf("base delay between git retry attempts (capped at %s)", CloneRetryDelayCap))

	return cmd
}

// RunClone executes the clone operation (exported for testing)
func RunClone(ctx context.Context, rawURI string, outputPath string, opts *CloneOptions) error {
	uri, err := normalizeGitURI(rawURI)
	if err != nil {
		return fmt.Errorf("invalid repository URI: %w", err)
	}

	destinationPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Disable git interactivity
	os.Setenv("GIT_TERMINAL_PROMPT", "0")

	// Setup authentication
	authArgs, err := setupAuthentication(ctx, uri, opts)
	if err != nil {
		return fmt.Errorf("setting up authentication: %w", err)
	}

	// Setup credential store for reliable auth across redirects (best-effort)
	cleanupCreds := setupCredentialStore(uri)
	defer cleanupCreds()

	// Setup SSH if provided
	cleanupSSH, err := setupSSHKey(opts.SSHKey)
	if err != nil {
		return fmt.Errorf("setting up SSH key: %w", err)
	}
	defer cleanupSSH()

	// Use temporary directory for cloning
	tmpPath, err := os.MkdirTemp(constants.DefaultTmpDirPath, "clone-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	// Ensure cleanup on any error
	defer func() {
		if err = os.RemoveAll(tmpPath); err != nil {
			err = fmt.Errorf("error cleaning up temporary directory: %w", err)
		}
	}()

	// Configure git settings
	configArgs := []string{
		"-c", fmt.Sprintf("safe.directory=%s", tmpPath),
		"-c", "advice.detachedHead=false",
	}

	// Clean paths for sparse checkout
	paths := cleanPaths(opts.RawPaths, opts.Cone)

	if !opts.isQuiet() {
		fmt.Printf("📦 ")
	}

	// Perform the clone operation
	if err := performClone(uri.String(), tmpPath, configArgs, authArgs, opts, paths); err != nil {
		return fmt.Errorf("error cloning repository: %w", err)
	}

	// Copy files to destination
	if err := copyRepositoryContents(tmpPath, destinationPath, opts); err != nil {
		return fmt.Errorf("error copying files to destination: %w", err)
	}

	// Adjust file permissions
	if err := adjustFilePermissions(destinationPath, opts); err != nil {
		return fmt.Errorf("error adjusting permissions: %w", err)
	}

	// List directory contents
	if err := listDirectoryContents(destinationPath, opts); err != nil {
		return fmt.Errorf("error listing directory contents: %w", err)
	}

	return nil
}

func (opts *CloneOptions) isQuiet() bool {
	return opts.Verbosity == "quiet"
}

// gitLogArgs returns git flags for the configured verbosity. Omit/unknown defaults to verbose.
func (opts *CloneOptions) gitLogArgs() []string {
	switch opts.Verbosity {
	case "quiet":
		return []string{"--quiet"}
	case "normal":
		return nil
	default:
		return []string{"--verbose"}
	}
}

func (opts *CloneOptions) retrySettings() (int, time.Duration) {
	count := CloneRetryOnFailureMaxAttempts
	delay := CloneRetryOnFailureBaseDelay
	if opts.RetryCount > 0 {
		count = opts.RetryCount
	}
	if opts.RetryDelay > 0 {
		delay = opts.RetryDelay
	}
	if count > CloneRetryCountCap {
		count = CloneRetryCountCap
	}
	if delay > CloneRetryDelayCap {
		delay = CloneRetryDelayCap
	}
	return count, delay
}

// normalizeGitURI normalizes a git URI by adding appropriate protocol if missing
func normalizeGitURI(rawURI string) (*url.URL, error) {
	// Convert SSH format (git@github.com:owner/repo.git) to URL format
	if !protocolRe.MatchString(rawURI) && strings.ContainsRune(rawURI, ':') && !strings.ContainsRune(rawURI, '\\') {
		rawURI = "ssh://" + strings.Replace(rawURI, ":", "/", 1)
	}
	return url.Parse(rawURI)
}

// setupAuthentication configures git authentication based on the auth type
func setupAuthentication(ctx context.Context, uri *url.URL, opts *CloneOptions) ([]string, error) {
	authArgs := make([]string, 0)

	switch opts.AuthType {
	case "header":
		ui.Debug("auth type: header")
		if opts.Token != "" {
			authArgs = append(authArgs, "-c", fmt.Sprintf("http.extraHeader=Authorization: Bearer %s", opts.Token))
		}
		if opts.Username != "" {
			uri.User = url.User(opts.Username)
		}

	case "github":
		ui.Debug("auth type: github")
		client, err := env.Cloud()
		if err != nil {
			return nil, fmt.Errorf("could not create cloud client: %w", err)
		}
		githubToken, err := client.GetGitHubToken(ctx, uri.String())
		if err == nil {
			// GitHub uses x-access-token as username for token auth
			uri.User = url.UserPassword("x-access-token", githubToken)
		}

	default: // basic auth
		ui.Debug("auth type: basic")
		if opts.Username != "" && opts.Token != "" {
			uri.User = url.UserPassword(opts.Username, opts.Token)
		} else if opts.Username != "" {
			uri.User = url.User(opts.Username)
		} else if opts.Token != "" {
			uri.User = url.User(opts.Token)
		}
	}

	return authArgs, nil
}

// setupSSHKey configures SSH authentication if an SSH key is provided and returns a cleanup function
func setupSSHKey(sshKey string) (func(), error) {
	if sshKey == "" {
		return func() {}, nil
	}

	// Ensure newline at EOF
	sshKey = strings.TrimRight(sshKey, "\n") + "\n"

	// Create temp file for SSH key
	tmpFile, err := os.CreateTemp(constants.DefaultTmpDirPath, "ssh-key-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file for SSH key: %w", err)
	}
	sshKeyPath := tmpFile.Name()
	tmpFile.Close() // Close it so we can write to it with proper permissions

	if err := os.WriteFile(sshKeyPath, []byte(sshKey), 0400); err != nil {
		_ = os.Remove(sshKeyPath)
		return nil, fmt.Errorf("writing SSH key: %w", err)
	}

	// Configure SSH command
	sshCmd := shellquote.Join("ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-i", sshKeyPath)
	os.Setenv("GIT_SSH_COMMAND", sshCmd)

	return func() { _ = os.Remove(sshKeyPath) }, nil
}

// setupCredentialStore configures a git credential store file so that credentials
// are available even when Git strips them from the URL (e.g., after HTTP redirects).
// This is best-effort: if anything fails, it logs a warning and returns a no-op
// cleanup, allowing the clone to proceed with URL-embedded credentials.
func setupCredentialStore(uri *url.URL) func() {
	noop := func() {}

	if uri.Scheme != "https" && uri.Scheme != "http" {
		return noop
	}

	// Use credentials from the URI (already set by setupAuthentication).
	// Skip credential-store setup when the URI only has a username and no
	// password/token, because that does not provide usable credentials and can
	// still alter global git authentication behavior.
	if uri.User == nil {
		return noop
	}
	if _, hasPassword := uri.User.Password(); !hasPassword {
		return noop
	}

	// Build the credential URL: scheme://user:token@host
	credURL := &url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
		User:   uri.User,
	}

	credContent := credURL.String() + "\n"

	// Write the credentials file
	credFile, err := os.CreateTemp(constants.DefaultTmpDirPath, "git-credentials-*")
	if err != nil {
		fmt.Printf("warn: could not create credential store file: %s\n", err)
		return noop
	}
	credPath := credFile.Name()
	credFile.Close()

	if err := os.WriteFile(credPath, []byte(credContent), 0400); err != nil {
		fmt.Printf("warn: could not write credential store: %s\n", err)
		_ = os.Remove(credPath)
		return noop
	}

	// Configure git to use the credential store
	if err := Run("git", "config", "--global", "credential.helper", shellquote.Join("store", "--file", credPath)); err != nil {
		fmt.Printf("warn: could not configure credential helper: %s\n", err)
		_ = os.Remove(credPath)
		return noop
	}

	return func() {
		if err := Run("git", "config", "--global", "--unset", "credential.helper"); err != nil {
			fmt.Printf("warn: could not unset credential helper: %s\n", err)
		}

		_ = os.Remove(credPath)
	}
}

// cleanPaths prepares paths for sparse checkout
func cleanPaths(rawPaths []string, cone bool) []string {
	paths := make([]string, 0, len(rawPaths))
	for _, p := range rawPaths {
		p = filepath.Clean(p)
		// In cone mode, remove leading slash except for root
		if cone && p != "/" && strings.HasPrefix(p, "/") {
			p = p[1:]
		}
		if p != "" && p != "." {
			paths = append(paths, p)
		}
	}
	return paths
}

// performClone executes the git clone operation
func performClone(uri, outputPath string, configArgs, authArgs []string, opts *CloneOptions, paths []string) error {
	if len(paths) == 0 {
		return performFullClone(uri, outputPath, configArgs, authArgs, opts)
	}
	return performSparseClone(uri, outputPath, configArgs, authArgs, opts, paths)
}

// performFullClone performs a full repository clone
func performFullClone(uri, outputPath string, configArgs, authArgs []string, opts *CloneOptions) error {
	ui.Debug("performing full checkout")

	retryCount, retryDelay := opts.retrySettings()
	args := []interface{}{"clone"}
	// Always clone without checkout so we can resolve `revision` (branch, tag, or commit)
	// using the same fetch+checkout logic as the sparse path.
	args = append(args, configArgs, authArgs, "--depth", "1", "--no-checkout", opts.gitLogArgs(), uri, outputPath)

	if err := RunWithRetry(retryCount, retryDelay, "git", args...); err != nil {
		return err
	}

	if err := checkoutRevision(outputPath, configArgs, authArgs, opts); err != nil {
		return fmt.Errorf("checking out revision: %w", err)
	}

	return nil
}

// performSparseClone performs a sparse repository clone
func performSparseClone(uri, outputPath string, configArgs, authArgs []string, opts *CloneOptions, paths []string) error {
	ui.Debug("performing sparse checkout")

	if err := initializeSparseRepo(uri, outputPath, configArgs, authArgs, opts); err != nil {
		return fmt.Errorf("initializing sparse repository: %w", err)
	}

	if err := configureSparseCheckout(outputPath, configArgs, paths, opts); err != nil {
		return fmt.Errorf("configuring sparse checkout: %w", err)
	}

	if err := checkoutRevision(outputPath, configArgs, authArgs, opts); err != nil {
		return fmt.Errorf("checking out revision: %w", err)
	}

	return nil
}

// initializeSparseRepo initializes a sparse git repository without checking out files
func initializeSparseRepo(uri, outputPath string, configArgs, authArgs []string, opts *CloneOptions) error {
	retryCount, retryDelay := opts.retrySettings()
	args := []interface{}{"clone"}
	args = append(args, configArgs, authArgs,
		"--filter=blob:none",
		"--no-checkout",
		"--sparse",
		"--depth", "1",
		opts.gitLogArgs(),
		uri, outputPath)

	return RunWithRetry(retryCount, retryDelay, "git", args...)
}

// configureSparseCheckout sets up sparse checkout patterns
func configureSparseCheckout(repoPath string, configArgs, paths []string, opts *CloneOptions) error {
	retryCount, retryDelay := opts.retrySettings()
	sparseArgs := []interface{}{"-C", repoPath}
	sparseArgs = append(sparseArgs, configArgs, "sparse-checkout", "set")

	if !opts.Cone {
		sparseArgs = append(sparseArgs, "--no-cone")
	}

	sparseArgs = append(sparseArgs, paths)

	return RunWithRetry(retryCount, retryDelay, "git", sparseArgs...)
}

// checkoutRevision checks out a specific revision or the default branch
func checkoutRevision(repoPath string, configArgs, authArgs []string, opts *CloneOptions) error {
	if opts.Revision == "" {
		// For sparse checkout, we need to populate the working directory
		// The sparse-checkout configuration will control which files are checked out
		return Run("git", "-C", repoPath, configArgs, "read-tree", "-m", "-u", "HEAD")
	}

	if err := fetchRevision(repoPath, configArgs, authArgs, opts); err != nil {
		return fmt.Errorf("fetching revision: %w", err)
	}

	if err := Run("git", "-C", repoPath, configArgs, "checkout", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checking out FETCH_HEAD: %w", err)
	}

	// Create branch for non-commit references
	// Skip branch creation if revision looks like a commit hash (40 hex chars)
	if !isCommitHash(opts.Revision) {
		if err := Run("git", "-C", repoPath, configArgs, "checkout", "-B", opts.Revision); err != nil {
			return fmt.Errorf("creating branch: %w", err)
		}
	}

	return nil
}

// fetchRevision fetches a specific revision from the remote
func fetchRevision(repoPath string, configArgs, authArgs []string, opts *CloneOptions) error {
	retryCount, retryDelay := opts.retrySettings()
	fetchArgs := []interface{}{"-C", repoPath}
	fetchArgs = append(fetchArgs, configArgs, "fetch", authArgs, "--depth", "1", opts.gitLogArgs(), "origin", opts.Revision)

	return RunWithRetry(retryCount, retryDelay, "git", fetchArgs...)
}

// isCommitHash checks if a string looks like a git commit hash
func isCommitHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// copyRepositoryContents copies repository contents from source to destination
func copyRepositoryContents(src, dest string, opts *CloneOptions) error {
	if !opts.isQuiet() {
		fmt.Printf("📥 Moving the contents to %s...\n", dest)
	}

	return copy.Copy(src, dest, copy.Options{
		OnError: func(srcPath, destPath string, err error) error {
			if err != nil {
				// Ignore chmod errors on mounted directories
				if srcPath == src && strings.Contains(err.Error(), "chmod") {
					return nil
				}
				fmt.Printf("warn: copying to %s: %s\n", destPath, err.Error())
			}
			return nil
		},
	})
}

// adjustFilePermissions ensures files have appropriate permissions
func adjustFilePermissions(path string, opts *CloneOptions) error {
	if !opts.isQuiet() {
		fmt.Printf("📥 Adjusting access permissions...\n")
	}

	return filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		mode := info.Mode()
		// Ensure group has read/write permissions
		if mode.Perm()&0o060 != 0o060 {
			if err := os.Chmod(filePath, mode|0o060); err != nil {
				// Log but don't fail on permission errors
				fmt.Printf("warn: chmod %s: %s\n", filePath, err.Error())
			}
		}
		return nil
	})
}

// listDirectoryContents displays the contents of a directory
func listDirectoryContents(path string, opts *CloneOptions) error {
	if opts.isQuiet() {
		return nil
	}

	fmt.Printf("🔎 Destination folder contains following files ...\n")

	return filepath.Walk(path, func(name string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Bold directory names
		if info.IsDir() {
			fmt.Printf("\x1b[1m%s\x1b[0m\n", name)
		} else {
			fmt.Println(name)
		}
		return nil
	})
}
