// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func mapSlice[T any, U any](s []T, fn func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = fn(s[i])
	}
	return result
}

func deduplicateRoots(paths []string) []string {
	result := make([]string, 0)
loop:
	for _, path := range paths {
		for _, path2 := range paths {
			if strings.HasPrefix(path, path2+"/") {
				continue loop
			}
		}
		result = append(result, path)
	}
	return result
}

func findSearchRoot(pattern string) string {
	path, _ := doublestar.SplitPattern(pattern + "/")
	return strings.TrimRight(path, "/")
}

// TODO: Support wildcards better:
//   - /**/*.json is a part of /data
//   - /data/s*me/*a*/abc.json is a part of /data/some/path/
func isPatternIn(pattern string, dirs []string) bool {
	return isPathIn(findSearchRoot(pattern), dirs)
}

func isPathIn(path string, dirs []string) bool {
	for _, dir := range dirs {
		path = strings.TrimRight(path, "/")
		if dir == path || strings.HasPrefix(path, dir+"/") {
			return true
		}
	}
	return false
}

func sanitizePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	path = strings.TrimRight(filepath.ToSlash(path), "/")
	if path == "" {
		path = "/"
	}
	return path, err
}

func sanitizePaths(input []string) ([]string, error) {
	paths := make([]string, len(input))
	for i := range input {
		var err error
		paths[i], err = sanitizePath(input[i])
		if err != nil {
			return nil, fmt.Errorf("error while resolving path: %s: %w", input[i], err)
		}
	}
	return paths, nil
}

func filterPatterns(patterns, dirs []string) []string {
	result := make([]string, 0)
	for _, p := range patterns {
		if isPatternIn(p, dirs) {
			result = append(result, p)
		}
	}
	return result
}

func detectCommonPath(path1, path2 string) string {
	if path1 == path2 {
		return path1
	}
	common := 0
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] != parts2[i] {
			break
		}
		common++
	}
	if common == 1 && parts1[0] == "" {
		return "/"
	}
	return strings.Join(parts1[0:common], "/")
}

func detectRoot(potential string, paths []string) string {
	potential = strings.TrimRight(potential, "/")
	if potential == "" {
		potential = "/"
	}
	for _, path := range paths {
		potential = detectCommonPath(potential, path)
	}
	return potential
}

func CreateWalker(patterns, roots []string, root string) (Walker, error) {
	var err error

	// Build absolute paths
	if patterns, err = sanitizePaths(patterns); err != nil {
		return nil, err
	}
	if roots, err = sanitizePaths(roots); err != nil {
		return nil, err
	}
	if root, err = sanitizePath(root); err != nil {
		return nil, err
	}
	// Include only if it is matching some mounted volumes
	patterns = filterPatterns(patterns, roots)
	// Detect top-level paths for searching
	searchPaths := deduplicateRoots(mapSlice(patterns, findSearchRoot))
	// Detect root path for the bucket
	root = detectRoot(root, searchPaths)

	return &walker{
		root:        root,
		searchPaths: searchPaths,
		patterns:    patterns,
	}, nil
}

type walker struct {
	root        string
	searchPaths []string
	patterns    []string // TODO: Optimize to check only patterns matching specific searchPaths
}

type WalkerFn = func(path string, file fs.File, err error) error

type Walker interface {
	Root() string
	SearchPaths() []string
	Patterns() []string
	Walk(fsys fs.FS, walker WalkerFn) error
}

func (w *walker) Root() string {
	return w.root
}

func (w *walker) SearchPaths() []string {
	return w.searchPaths
}

func (w *walker) Patterns() []string {
	return w.patterns
}

func (w *walker) matches(filePath string) bool {
	for _, p := range w.patterns {
		v, _ := doublestar.PathMatch(p, filePath)
		if v {
			return true
		}
	}
	return false
}

func (w *walker) walk(fsys fs.FS, path string, walker WalkerFn) error {
	sanitizedPath := strings.TrimLeft(path, "/")
	if sanitizedPath == "" {
		sanitizedPath = "."
	}

	return fs.WalkDir(fsys, sanitizedPath, func(filePath string, d fs.DirEntry, err error) error {
		resolvedPath := "/" + filepath.ToSlash(filePath)
		if !w.matches(resolvedPath) {
			return nil
		}
		if err != nil {
			fmt.Printf("Warning: '%s' ignored from scraping: %v\n", resolvedPath, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		file, err := fsys.Open(filePath)
		return walker(strings.TrimLeft(resolvedPath[len(w.root):], "/"), file, err)
	})
}

func (w *walker) Walk(fsys fs.FS, walker WalkerFn) (err error) {
	for _, s := range w.searchPaths {
		err = w.walk(fsys, s, walker)
		if err != nil {
			return err
		}
	}
	return nil
}
