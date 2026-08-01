// Package matchers exports pure predicate helpers for evaluating whether a git
// event should trigger a workflow. Callers pass primitive slices and strings
// so the package stays free of provider-specific and control-plane-specific
// types, making it consumable both by the OSS TestTrigger informer and by
// external tools that need to reason about the same rules.
package matchers

import (
	"path/filepath"
	"strings"
)

// NormalizePaths trims whitespace and surrounding slashes from each path and
// drops empty entries. Returns nil if the input is empty or entirely blank.
func NormalizePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.Trim(strings.TrimSpace(p), "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// PathMatches reports whether file matches any of paths. Paths are normalized
// before matching (whitespace and surrounding slashes trimmed).
func PathMatches(paths []string, file string) bool {
	return PathMatchesNormalized(NormalizePaths(paths), file)
}

// PathMatchesNormalized reports whether file matches any of paths. Paths must
// already be normalized (see NormalizePaths).
func PathMatchesNormalized(paths []string, file string) bool {
	for _, p := range paths {
		if MatchGlob(p, file) {
			return true
		}
	}
	return false
}

// PathIsIgnored returns true if file matches any of the ignore patterns.
func PathIsIgnored(ignorePatterns []string, file string) bool {
	if len(ignorePatterns) == 0 {
		return false
	}
	for _, p := range ignorePatterns {
		if MatchGlob(p, file) {
			return true
		}
	}
	return false
}

// BranchFromRef extracts the branch name from a full ref like
// "refs/heads/main". Returns "" if ref is not a branch ref.
func BranchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ""
}

// TagFromRef extracts the tag name from a full ref like "refs/tags/v1.0.0".
// Returns "" if ref is not a tag ref.
func TagFromRef(ref string) string {
	const prefix = "refs/tags/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ""
}

// MatchGlob performs glob-style matching supporting * and ** patterns.
// It matches file paths against patterns like "src/**", "*.md", "docs/*".
// Malformed patterns are treated as non-matching (filepath.Match returns
// ErrBadPattern).
func MatchGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	if strings.Contains(pattern, "**") {
		pattern = filepath.ToSlash(strings.TrimSuffix(pattern, "/"))
		name = filepath.ToSlash(strings.TrimSuffix(name, "/"))

		pSegs := strings.Split(pattern, "/")
		nSegs := strings.Split(name, "/")

		type state struct{ i, j int }
		memo := map[state]bool{}
		var match func(i, j int) bool
		match = func(i, j int) bool {
			s := state{i, j}
			if v, ok := memo[s]; ok {
				return v
			}

			var res bool
			switch {
			case i == len(pSegs):
				res = j == len(nSegs)
			case pSegs[i] == "**":
				res = match(i+1, j) || (j < len(nSegs) && match(i, j+1))
			case j < len(nSegs):
				ok, err := filepath.Match(pSegs[i], nSegs[j])
				res = err == nil && ok && match(i+1, j+1)
			default:
				res = false
			}

			memo[s] = res
			return res
		}
		return match(0, 0)
	}

	normalizedPattern := strings.TrimSuffix(pattern, "/")
	if name == normalizedPattern || strings.HasPrefix(name, normalizedPattern+"/") {
		return true
	}
	return false
}

// NameMatchesPatterns checks if a name (branch or tag) matches any of the given
// glob patterns. Returns true if patterns is empty (matches all).
func NameMatchesPatterns(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	return NameMatchesAny(name, patterns)
}

// NameMatchesAny checks if a name matches any of the given glob patterns.
// Returns false if patterns is empty.
func NameMatchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		matched, err := filepath.Match(p, name)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// PRMatchesBaseBranch checks if a PR's base branch matches the branch filters.
// branches is the allowlist (empty means match-all); branchesIgnore is the
// denylist (takes precedence).
func PRMatchesBaseBranch(baseBranch string, branches, branchesIgnore []string) bool {
	if NameMatchesAny(baseBranch, branchesIgnore) {
		return false
	}
	if len(branches) == 0 {
		return true
	}
	return NameMatchesAny(baseBranch, branches)
}

// PRMatchesTypes checks if a PR action matches the configured types filter.
// Returns true when types is empty (matches all).
func PRMatchesTypes(action string, types []string) bool {
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		if strings.EqualFold(strings.TrimSpace(t), action) {
			return true
		}
	}
	return false
}

// DeterminePRAction determines the PR action based on state transitions.
// prevEncoded and currentEncoded are "sha:state" strings; headSHA is the
// current head SHA of the PR.
func DeterminePRAction(prevEncoded, currentEncoded, headSHA string) string {
	prevParts := strings.SplitN(prevEncoded, ":", 2)
	currParts := strings.SplitN(currentEncoded, ":", 2)

	prevSHA := ""
	prevState := ""
	if len(prevParts) == 2 {
		prevSHA = prevParts[0]
		prevState = prevParts[1]
	}

	currState := ""
	if len(currParts) == 2 {
		currState = currParts[1]
	}

	if currState == "closed" && prevState != "closed" {
		return "closed"
	}
	if currState == "open" && prevState == "closed" {
		return "reopened"
	}
	if prevSHA != headSHA {
		return "synchronize"
	}
	return "synchronize"
}

// PRPathsMatch checks if any changed file in a PR matches the path filters.
// If paths is empty, any non-ignored file matches. pathsIgnore takes precedence
// per-file.
func PRPathsMatch(changedFiles, paths, pathsIgnore []string) bool {
	for _, file := range changedFiles {
		if PathIsIgnored(pathsIgnore, file) {
			continue
		}
		if len(paths) == 0 || PathMatchesNormalized(paths, file) {
			return true
		}
	}
	return false
}
