// Package localrunner contains the local, Kubernetes-backed TestWorkflow runner.
package localrunner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SourceOptions controls the source tree written to the local runner's relay
// archive. Patterns use the deliberately small Testkube ignore syntax: *, ?,
// character classes, and ** are supported; a pattern without a slash matches
// any path component. This is not a complete .gitignore implementation.
type SourceOptions struct {
	Directory string
	Includes  []string
	Excludes  []string
	MaxBytes  int64
}

// SourceSummary describes the uncompressed source data written to an archive.
// Bytes counts regular-file payload bytes. Files counts regular-file and
// symlink entries, but not directory entries.
type SourceSummary struct {
	Bytes int64
	Files int
}

// WriteSourceArchive streams a gzip-compressed tar archive of opts.Directory
// to dst. It never creates a local archive file. The caller must provide a
// positive MaxBytes limit; the limit applies to uncompressed regular-file
// payloads, rather than the compressed archive size.
//
// The walker deliberately visits ignored directories too. That permits a
// later ! rule in .testkubeignore, or an explicit source include, to restore a
// file below an otherwise excluded directory.
func WriteSourceArchive(ctx context.Context, opts SourceOptions, dst io.Writer) (SourceSummary, error) {
	if ctx == nil {
		return SourceSummary{}, fmt.Errorf("source archive context is required")
	}
	if err := ctx.Err(); err != nil {
		return SourceSummary{}, err
	}
	if opts.MaxBytes <= 0 {
		return SourceSummary{}, fmt.Errorf("max source bytes must be greater than zero")
	}
	if dst == nil {
		return SourceSummary{}, fmt.Errorf("source archive destination is required")
	}

	root, err := sourceRoot(opts.Directory)
	if err != nil {
		return SourceSummary{}, err
	}
	rules, err := sourceRules(root, opts)
	if err != nil {
		return SourceSummary{}, err
	}

	gzipWriter := gzip.NewWriter(dst)
	tarWriter := tar.NewWriter(gzipWriter)
	var summary SourceSummary

	walkErr := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("read source entry %q: %w", filePath, walkErr)
		}
		if filePath == root {
			return nil
		}

		name, err := sourceArchiveName(root, filePath)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat source entry %q: %w", name, err)
		}

		// Validate every symlink in the source tree before filtering it. An
		// ignored unsafe link must not make a later explicit include unsafe.
		linkName := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkName, err = safeSymlinkTarget(root, filePath, name)
			if err != nil {
				return err
			}
		}

		include, err := shouldIncludeSource(name, info.IsDir(), rules)
		if err != nil {
			return err
		}
		if !include {
			return nil
		}

		switch {
		case info.IsDir():
			return writeSourceHeader(tarWriter, info, name, "")
		case info.Mode()&os.ModeSymlink != 0:
			if err := writeSourceHeader(tarWriter, info, name, linkName); err != nil {
				return err
			}
			summary.Files++
			return nil
		case info.Mode().IsRegular():
			return writeSourceFile(ctx, tarWriter, root, filePath, name, opts.MaxBytes, &summary)
		default:
			return fmt.Errorf("unsupported source entry type for %q", name)
		}
	})

	// Closing both writers is needed even after a walk error so a blocking
	// consumer is not left waiting for a close. The walk error remains the
	// useful error if the archive is intentionally incomplete.
	tarCloseErr := tarWriter.Close()
	gzipCloseErr := gzipWriter.Close()
	if walkErr != nil {
		return SourceSummary{}, walkErr
	}
	if tarCloseErr != nil {
		return SourceSummary{}, fmt.Errorf("finish source tar archive: %w", tarCloseErr)
	}
	if gzipCloseErr != nil {
		return SourceSummary{}, fmt.Errorf("finish source gzip archive: %w", gzipCloseErr)
	}
	if err := ctx.Err(); err != nil {
		return SourceSummary{}, err
	}
	return summary, nil
}

func sourceRoot(directory string) (string, error) {
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("source directory is required")
	}
	abs, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve source directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve source directory %q: %w", directory, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat source directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source path %q is not a directory", directory)
	}
	return resolved, nil
}

func writeSourceHeader(writer *tar.Writer, info fs.FileInfo, name, linkName string) error {
	header, err := tar.FileInfoHeader(info, linkName)
	if err != nil {
		return fmt.Errorf("build source archive header for %q: %w", name, err)
	}
	header.Name = name
	if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("write source archive header for %q: %w", name, err)
	}
	return nil
}

func writeSourceFile(
	ctx context.Context,
	writer *tar.Writer,
	root, filePath, name string,
	maxBytes int64,
	summary *SourceSummary,
) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", name, err)
	}
	defer file.Close()

	before, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat open source file %q: %w", name, err)
	}
	pathInfo, err := os.Lstat(filePath)
	if err != nil {
		return fmt.Errorf("re-stat source file %q: %w", name, err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, pathInfo) {
		return fmt.Errorf("source file %q changed while packaging", name)
	}
	if !before.Mode().IsRegular() {
		return fmt.Errorf("source entry %q changed to an unsupported type while packaging", name)
	}
	if before.Size() < 0 || before.Size() > maxBytes-summary.Bytes {
		return fmt.Errorf("source archive exceeds max source bytes (%d) while adding %q", maxBytes, name)
	}
	if err := writeSourceHeader(writer, before, name, ""); err != nil {
		return err
	}
	if err := copySourceFile(ctx, writer, file, before.Size()); err != nil {
		return fmt.Errorf("write source file %q: %w", name, err)
	}

	after, err := file.Stat()
	if err != nil {
		return fmt.Errorf("re-stat source file %q: %w", name, err)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return fmt.Errorf("source file %q changed while packaging", name)
	}
	resolved, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return fmt.Errorf("re-resolve source file %q: %w", name, err)
	}
	if !isInsideSourceRoot(root, resolved) {
		return fmt.Errorf("source file %q escapes the source root", name)
	}

	summary.Bytes += before.Size()
	summary.Files++
	return nil
}

func copySourceFile(ctx context.Context, dst io.Writer, src io.Reader, size int64) error {
	const bufferSize = 32 * 1024
	buffer := make([]byte, bufferSize)
	remaining := size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		n, readErr := src.Read(buffer[:readSize])
		if n > 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			written, writeErr := dst.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
			remaining -= int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF && remaining == 0 {
				break
			}
			if readErr == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return readErr
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func sourceArchiveName(root, filePath string) (string, error) {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return "", fmt.Errorf("build source archive path for %q: %w", filePath, err)
	}
	name := filepath.ToSlash(relative)
	if !safeArchiveName(name) {
		return "", fmt.Errorf("unsafe source archive entry name %q", name)
	}
	return name, nil
}

func safeArchiveName(name string) bool {
	if name == "" || name == "." || strings.ContainsRune(name, 0) || filepath.IsAbs(name) || path.IsAbs(name) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return !strings.HasPrefix(name, "../")
}

func safeSymlinkTarget(root, linkPath, archiveName string) (string, error) {
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("read source symlink %q: %w", archiveName, err)
	}
	if target == "" || strings.ContainsRune(target, 0) {
		return "", fmt.Errorf("unsafe empty source symlink target for %q", archiveName)
	}
	// Backslashes are path separators on Windows. Treating them as separators
	// here keeps the emitted Unix tar safe when it is later unpacked there.
	portableTarget := strings.ReplaceAll(target, "\\", "/")
	if filepath.IsAbs(target) || path.IsAbs(portableTarget) {
		return "", fmt.Errorf("unsafe absolute symlink target for %q", archiveName)
	}
	for _, part := range strings.Split(portableTarget, "/") {
		if part == ".." {
			return "", fmt.Errorf("unsafe parent-traversing symlink target for %q", archiveName)
		}
	}

	resolved, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(linkPath), target))
	if err != nil {
		return "", fmt.Errorf("resolve source symlink target for %q: %w", archiveName, err)
	}
	if !isInsideSourceRoot(root, resolved) {
		return "", fmt.Errorf("unsafe escaping symlink target for %q", archiveName)
	}
	return portableTarget, nil
}

func isInsideSourceRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

var defaultSourceExclusions = []string{
	".git/",
	".testkube/",
	"node_modules/",
	"vendor/",
	"dist/",
	"build/",
	"coverage/",
	".coverage/",
	".idea/",
	".vscode/",
	".vs/",
	".DS_Store",
	"Thumbs.db",
	"Desktop.ini",
	"*~",
	"*.swp",
	"*.swo",
	"#*#",
}

type sourceRule struct {
	pattern       string
	negated       bool
	directoryOnly bool
	hasSlash      bool
}

func sourceRules(root string, opts SourceOptions) ([]sourceRule, error) {
	rules := make([]sourceRule, 0, len(defaultSourceExclusions)+len(opts.Includes)+len(opts.Excludes))
	for _, pattern := range defaultSourceExclusions {
		rule, err := parseSourceRule(pattern, false)
		if err != nil {
			return nil, fmt.Errorf("parse built-in source exclusion %q: %w", pattern, err)
		}
		rules = append(rules, rule)
	}

	ignoreRules, err := readTestkubeIgnore(root)
	if err != nil {
		return nil, err
	}
	rules = append(rules, ignoreRules...)

	for _, pattern := range opts.Excludes {
		rule, err := parseFlagSourceRule(pattern, false, "source exclude")
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	for _, pattern := range opts.Includes {
		rule, err := parseFlagSourceRule(pattern, true, "source include")
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func readTestkubeIgnore(root string) ([]sourceRule, error) {
	ignorePath := filepath.Join(root, ".testkubeignore")
	contents, err := os.ReadFile(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read .testkubeignore: %w", err)
	}

	lines := strings.Split(string(contents), "\n")
	rules := make([]sourceRule, 0, len(lines))
	for lineNumber, line := range lines {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		negated := strings.HasPrefix(line, "!")
		if negated {
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
		}
		rule, err := parseSourceRule(line, negated)
		if err != nil {
			return nil, fmt.Errorf("parse .testkubeignore line %d: %w", lineNumber+1, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func parseFlagSourceRule(pattern string, include bool, label string) (sourceRule, error) {
	if strings.HasPrefix(strings.TrimSpace(pattern), "!") {
		return sourceRule{}, fmt.Errorf("%s pattern %q must not begin with a negation marker", label, pattern)
	}
	rule, err := parseSourceRule(pattern, include)
	if err != nil {
		return sourceRule{}, fmt.Errorf("parse %s pattern %q: %w", label, pattern, err)
	}
	return rule, nil
}

func parseSourceRule(raw string, negated bool) (sourceRule, error) {
	pattern := strings.TrimSpace(raw)
	if pattern == "" {
		return sourceRule{}, fmt.Errorf("pattern is empty")
	}
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	directoryOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = strings.TrimPrefix(pattern, "/")
	if pattern == "" || strings.ContainsRune(pattern, 0) || path.IsAbs(pattern) {
		return sourceRule{}, fmt.Errorf("pattern %q is unsafe", raw)
	}
	for _, component := range strings.Split(pattern, "/") {
		if component == "" || component == "." || component == ".." {
			return sourceRule{}, fmt.Errorf("pattern %q is unsafe", raw)
		}
		if _, err := path.Match(component, ""); err != nil && component != "**" {
			return sourceRule{}, fmt.Errorf("pattern %q is invalid: %w", raw, err)
		}
	}
	return sourceRule{
		pattern:       pattern,
		negated:       negated,
		directoryOnly: directoryOnly,
		hasSlash:      strings.Contains(pattern, "/"),
	}, nil
}

func shouldIncludeSource(name string, isDir bool, rules []sourceRule) (bool, error) {
	included := true
	for _, rule := range rules {
		matches, err := rule.matches(name, isDir)
		if err != nil {
			return false, err
		}
		if matches {
			included = rule.negated
		}
	}
	return included, nil
}

func (rule sourceRule) matches(name string, isDir bool) (bool, error) {
	parts := strings.Split(name, "/")
	if rule.directoryOnly {
		for index := 1; index <= len(parts); index++ {
			matched, err := rule.matchesPath(strings.Join(parts[:index], "/"))
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}

	// A non-directory pattern can still match a directory itself. Its
	// descendants are handled naturally by a component-only pattern such as
	// "node_modules" or by an explicit ** pattern.
	_ = isDir
	return rule.matchesPath(name)
}

func (rule sourceRule) matchesPath(candidate string) (bool, error) {
	if !rule.hasSlash {
		for _, component := range strings.Split(candidate, "/") {
			matched, err := path.Match(rule.pattern, component)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}
	return matchDoubleStar(strings.Split(rule.pattern, "/"), strings.Split(candidate, "/"))
}

func matchDoubleStar(pattern, candidate []string) (bool, error) {
	if len(pattern) == 0 {
		return len(candidate) == 0, nil
	}
	if pattern[0] == "**" {
		for offset := 0; offset <= len(candidate); offset++ {
			matched, err := matchDoubleStar(pattern[1:], candidate[offset:])
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}
	if len(candidate) == 0 {
		return false, nil
	}
	matched, err := path.Match(pattern[0], candidate[0])
	if err != nil || !matched {
		return matched, err
	}
	return matchDoubleStar(pattern[1:], candidate[1:])
}
