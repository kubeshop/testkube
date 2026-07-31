package matchers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/git/matchers"
)

func TestPathMatches(t *testing.T) {
	cases := []struct {
		paths []string
		file  string
		want  bool
	}{
		{[]string{"src"}, "src/main.go", true},
		{[]string{"src"}, "src", true},
		{[]string{"src/"}, "src/main.go", true},
		{[]string{"other"}, "src/main.go", false},
		{[]string{"src", "pkg"}, "pkg/util.go", true},
		{[]string{""}, "anything.go", false},
		{[]string{"src/sub"}, "src/sub/file.go", true},
		{[]string{"src/sub"}, "src/other/file.go", false},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.PathMatches(tc.paths, tc.file))
		})
	}
}

func TestPathMatchesNormalized(t *testing.T) {
	paths := matchers.NormalizePaths([]string{"src/", " pkg "})
	require.True(t, matchers.PathMatchesNormalized(paths, "src/main.go"))
	require.True(t, matchers.PathMatchesNormalized(paths, "pkg/util.go"))
	require.False(t, matchers.PathMatchesNormalized(paths, "internal/main.go"))
}

func TestNormalizePaths(t *testing.T) {
	paths := []string{" /a ", "/b/c", "", "///", "d/"}
	require.Equal(t, []string{"a", "b/c", "d"}, matchers.NormalizePaths(paths))
}

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"exact match", "main.go", "main.go", true},
		{"no match", "main.go", "other.go", false},
		{"star matches extension", "*.go", "main.go", true},
		{"doublestar matches nested", "src/**/*.go", "src/pkg/main.go", true},
		{"doublestar matches multiple dirs", "src/**/*.go", "src/a/b/c.go", true},
		{"doublestar no match outside prefix", "src/**/*.go", "pkg/main.go", false},
		{"prefix directory match", "src", "src/main.go", true},
		{"prefix directory exact", "src", "src", true},
		{"prefix directory trailing slash", "src/", "src/main.go", true},
		{"question mark single char", "?.go", "a.go", true},
		{"question mark no match multi char", "?.go", "ab.go", false},
		{"doublestar prefix only", "src/**", "src/a/b.go", true},
		{"doublestar suffix only", "**/test.go", "a/b/test.go", true},
		{"malformed pattern returns false", "[invalid", "anything", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.MatchGlob(tc.pattern, tc.input))
		})
	}
}

func TestNameMatchesPatterns(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		patterns []string
		want     bool
	}{
		{"empty patterns matches all", "main", nil, true},
		{"exact match", "main", []string{"main"}, true},
		{"no match", "develop", []string{"main"}, false},
		{"glob star", "feature/foo", []string{"feature/*"}, true},
		{"glob star no match", "bugfix/foo", []string{"feature/*"}, false},
		{"multiple patterns first matches", "main", []string{"main", "develop"}, true},
		{"multiple patterns second matches", "develop", []string{"main", "develop"}, true},
		{"whitespace trimmed", "main", []string{"  main  "}, true},
		{"empty pattern skipped", "main", []string{""}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.NameMatchesPatterns(tc.input, tc.patterns))
		})
	}
}

func TestNameMatchesAny(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		patterns []string
		want     bool
	}{
		{"empty patterns returns false", "main", nil, false},
		{"exact match", "main", []string{"main"}, true},
		{"glob match", "v1.0.0", []string{"v*"}, true},
		{"no match", "develop", []string{"main", "release/*"}, false},
		{"glob question mark", "v1", []string{"v?"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.NameMatchesAny(tc.input, tc.patterns))
		})
	}
}

func TestPathIsIgnored(t *testing.T) {
	cases := []struct {
		name     string
		patterns []string
		file     string
		want     bool
	}{
		{"empty patterns", nil, "any/file.go", false},
		{"matches glob", []string{"*.md"}, "README.md", true},
		{"does not match", []string{"*.md"}, "main.go", false},
		{"directory pattern", []string{"docs"}, "docs/guide.md", true},
		{"multiple patterns second matches", []string{"*.txt", "*.md"}, "notes.md", true},
		{"doublestar ignore", []string{"**/*.test.js"}, "src/deep/file.test.js", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.PathIsIgnored(tc.patterns, tc.file))
		})
	}
}

func TestBranchFromRef(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/foo", "feature/foo"},
		{"refs/tags/v1.0", ""},
		{"HEAD", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.ref, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.BranchFromRef(tc.ref))
		})
	}
}

func TestTagFromRef(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/tags/v1.0", "v1.0"},
		{"refs/tags/release/2.0", "release/2.0"},
		{"refs/heads/main", ""},
		{"HEAD", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.ref, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.TagFromRef(tc.ref))
		})
	}
}

func TestPRMatchesBaseBranch(t *testing.T) {
	cases := []struct {
		name           string
		base           string
		branches       []string
		branchesIgnore []string
		want           bool
	}{
		{"nil config matches all", "main", nil, nil, true},
		{"empty branches matches all", "main", nil, nil, true},
		{"branch matches", "main", []string{"main", "develop"}, nil, true},
		{"branch does not match", "feature/x", []string{"main"}, nil, false},
		{"glob match", "release/1.0", []string{"release/*"}, nil, true},
		{"ignore takes precedence", "main", []string{"main"}, []string{"main"}, false},
		{"ignore only", "main", nil, []string{"main"}, false},
		{"ignore does not match", "develop", nil, []string{"main"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.PRMatchesBaseBranch(tc.base, tc.branches, tc.branchesIgnore))
		})
	}
}

func TestPRMatchesTypes(t *testing.T) {
	cases := []struct {
		name   string
		action string
		types  []string
		want   bool
	}{
		{"nil types matches all", "opened", nil, true},
		{"empty types matches all", "synchronize", []string{}, true},
		{"type matches", "opened", []string{"opened", "synchronize"}, true},
		{"type does not match", "closed", []string{"opened"}, false},
		{"case insensitive", "Opened", []string{"opened"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.PRMatchesTypes(tc.action, tc.types))
		})
	}
}

func TestDeterminePRAction(t *testing.T) {
	const headSHA = "new-sha"
	cases := []struct {
		name    string
		prev    string
		current string
		want    string
	}{
		{"sha change is synchronize", "old-sha:open", "new-sha:open", "synchronize"},
		{"state to closed", "old-sha:open", "new-sha:closed", "closed"},
		{"state to open from closed", "old-sha:closed", "new-sha:open", "reopened"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.DeterminePRAction(tc.prev, tc.current, headSHA))
		})
	}
}

func TestPRPathsMatch(t *testing.T) {
	cases := []struct {
		name   string
		files  []string
		paths  []string
		ignore []string
		want   bool
	}{
		{"no filters matches all", []string{"src/main.go"}, nil, nil, true},
		{"path matches", []string{"src/main.go", "docs/readme.md"}, []string{"src/**"}, nil, true},
		{"path does not match", []string{"docs/readme.md"}, []string{"src/**"}, nil, false},
		{"ignore takes precedence", []string{"src/vendor/lib.go"}, []string{"src/**"}, []string{"src/vendor/**"}, false},
		{"mixed match after ignore", []string{"src/vendor/lib.go", "src/main.go"}, []string{"src/**"}, []string{"src/vendor/**"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, matchers.PRPathsMatch(tc.files, tc.paths, tc.ignore))
		})
	}
}
