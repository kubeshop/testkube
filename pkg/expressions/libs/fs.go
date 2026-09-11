package libs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func absPath(p, workingDir string) string {
	if !filepath.IsAbs(p) {
		p = filepath.Join(workingDir, p)
	}
	v, err := filepath.Abs(p)
	if err != nil {
		v = p
	}
	v = strings.TrimRight(filepath.ToSlash(v), "/")
	if v == "" {
		return "/"
	}
	return v
}

func findSearchRoot(pattern, workingDir string) string {
	path, _ := doublestar.SplitPattern(pattern + "/")
	path = strings.TrimRight(path, "/")
	if path == "." {
		return strings.TrimLeft(absPath("", workingDir), "/")
	}
	return strings.TrimLeft(path, "/")
}

func mapSlice[T any, U any](s []T, fn func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = fn(s[i])
	}
	return result
}

func deduplicateRoots(paths []string) []string {
	result := make([]string, 0)
	unique := make(map[string]struct{})
	for _, p := range paths {
		unique[p] = struct{}{}
	}
loop:
	for path := range unique {
		for path2 := range unique {
			if strings.HasPrefix(path, path2+"/") {
				continue loop
			}
		}
		result = append(result, path)
	}
	return result
}

func readFile(fsys fs.FS, workingDir string, values ...expressions.StaticValue) (interface{}, error) {
	if len(values) != 1 {
		return nil, errors.New("file() function takes a single argument")
	}
	if !values[0].IsString() {
		return nil, fmt.Errorf("file() function expects a string argument, provided: %v", values[0].String())
	}
	filePath, _ := values[0].StringValue()
	file, err := fsys.Open(strings.TrimLeft(absPath(filePath, workingDir), "/"))
	if err != nil {
		return nil, fmt.Errorf("opening file(%s): %w", filePath, err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file(%s): %w", filePath, err)
	}
	return string(content), nil
}

// hashFiles hashes the *contents* of every file matching the patterns.
//
// This exists because hash(glob(...)) hashes the list of paths instead, which
// changes when a file is added or removed but not when one is edited - a cache key
// built that way silently never invalidates. Hashing contents is almost always what
// the caller meant, and for more than one lockfile it is the only correct form.
func hashFiles(fsys fs.FS, workingDir string, values ...expressions.StaticValue) (interface{}, error) {
	matches, err := globFs(fsys, workingDir, values...)
	if err != nil {
		return nil, fmt.Errorf("hash_files(): %w", err)
	}
	paths, _ := matches.([]string)

	// globFs walks one root at a time, so its order follows the patterns. Sort, or the
	// same set of files would hash differently depending on how it was matched.
	sort.Strings(paths)

	if len(paths) == 0 {
		// No match is not an error: a lockfile that does not exist yet is a normal
		// state for a workflow being written. Callers that need a real key treat an
		// empty result as "do not cache" rather than failing the step.
		return "", nil
	}

	h := sha256.New()
	for _, path := range paths {
		file, err := fsys.Open(strings.TrimLeft(path, "/"))
		if err != nil {
			return nil, fmt.Errorf("hash_files(): opening %s: %w", path, err)
		}
		contents := sha256.New()
		_, err = io.Copy(contents, file)
		_ = file.Close()
		if err != nil {
			return nil, fmt.Errorf("hash_files(): reading %s: %w", path, err)
		}
		// Fold the path in too, so that moving a file changes the result.
		_, _ = h.Write([]byte(path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(contents.Sum(nil))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func createGlobMatcher(patterns []string) func(string) bool {
	return func(filePath string) bool {
		for _, p := range patterns {
			v, _ := doublestar.PathMatch(p, filePath)
			if v {
				return true
			}
		}
		return false
	}
}

func globFs(fsys fs.FS, workingDir string, values ...expressions.StaticValue) (interface{}, error) {
	if len(values) == 0 {
		return nil, errors.New("glob() function takes at least one argument")
	}

	// Read all the patterns
	ignorePatterns := make([]string, 0)
	patterns := make([]string, 0)
	for i := 0; i < len(values); i++ {
		v, _ := values[i].StringValue()
		if strings.HasPrefix(v, "!") {
			ignorePatterns = append(ignorePatterns, absPath(v[1:], workingDir))
		} else {
			patterns = append(patterns, absPath(v, workingDir))
		}
	}
	if len(patterns) == 0 {
		return nil, errors.New("glob() function needs at least one matching pattern")
	}
	matchesPositive := createGlobMatcher(patterns)
	matchesIgnore := createGlobMatcher(ignorePatterns)

	// Determine roots for searching, to avoid scanning whole FS
	findRoot := func(pattern string) string {
		return findSearchRoot(pattern, workingDir)
	}
	roots := deduplicateRoots(mapSlice(patterns, findRoot))
	result := make([]string, 0)
	for _, root := range roots {
		err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
			if path == "." || err != nil || d.IsDir() {
				return nil
			}
			path = "/" + path
			if !matchesPositive(path) || matchesIgnore(path) {
				return nil
			}
			result = append(result, path)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("glob() error: %v", err)
		}
	}

	return result, nil
}

func NewFsMachine(fsys fs.FS, workingDir string) expressions.Machine {
	if workingDir == "" {
		workingDir = "/"
	}
	return expressions.NewMachine().
		Register("workingDir", workingDir).
		RegisterFunction("file", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			v, err := readFile(fsys, workingDir, values...)
			return v, true, err
		}).
		RegisterFunction("glob", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			v, err := globFs(fsys, workingDir, values...)
			return v, true, err
		}).
		RegisterFunction("hash_files", func(values ...expressions.StaticValue) (interface{}, bool, error) {
			v, err := hashFiles(fsys, workingDir, values...)
			return v, true, err
		})
}
