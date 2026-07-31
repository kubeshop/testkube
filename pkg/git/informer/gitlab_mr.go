package informer

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
)

const (
	// gitlabAcceptJSON is sent on every GitLab REST call.
	gitlabAcceptJSON = "application/json"

	// gitlabMRListPageSize is GitLab's documented maximum page size. GitLab bumps
	// updated_at for more activity than GitHub does (approvals, pipeline results,
	// system notes), so the "most recently updated" window is noisier and a larger
	// page reduces the chance of evicting a merge request that received code.
	gitlabMRListPageSize = 100

	// gitlabMRDiffPageSize is deliberately capped at 30: the merge request diffs
	// endpoint returns HTTP 500 for per_page above 30.
	// See https://gitlab.com/gitlab-org/gitlab/-/issues/428187
	gitlabMRDiffPageSize = 30

	// gitlabMRDiffMaxPages bounds path-filter work for very large merge requests.
	gitlabMRDiffMaxPages = 4
)

// GitLab merge request states.
const (
	gitlabStateOpened = "opened"
	gitlabStateLocked = "locked"
	gitlabStateClosed = "closed"
	gitlabStateMerged = "merged"
)

// gitlabMR is a minimal GitLab merge request from the REST API v4 list endpoint.
type gitlabMR struct {
	IID          int       `json:"iid"`
	State        string    `json:"state"`
	Title        string    `json:"title"`
	UpdatedAt    time.Time `json:"updated_at"`
	WebURL       string    `json:"web_url"`
	SourceBranch string    `json:"source_branch"`
	TargetBranch string    `json:"target_branch"`
	SHA          string    `json:"sha"`
	Draft        bool      `json:"draft"`
	// Author is a value rather than a pointer so that the "author": null GitLab
	// returns for deleted users decodes to the zero value instead of needing a
	// nil check.
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
}

// gitlabMRDiff is one changed file in a GitLab merge request diff.
type gitlabMRDiff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	NewFile     bool   `json:"new_file"`
}

// gitlabProvider polls the GitLab REST API v4 for merge requests.
type gitlabProvider struct {
	apiBase     string
	projectPath string
	token       string
}

func newGitLabProvider(apiBase, projectPath, token string) *gitlabProvider {
	return &gitlabProvider{apiBase: apiBase, projectPath: projectPath, token: token}
}

func (p *gitlabProvider) kind() providerKind { return providerGitLab }

// headRef returns the ref GitLab publishes for a merge request head.
//
// Note that GitLab deletes this ref 14 days after a merge request is closed or
// merged, so TESTKUBE_GIT_PR_HEAD_SHA is the durable identifier for old merge
// requests.
func (p *gitlabProvider) headRef(iid int) string {
	return "refs/merge-requests/" + strconv.Itoa(iid) + "/head"
}

func (p *gitlabProvider) listPageSize() int { return gitlabMRListPageSize }

// list fetches the most recently updated merge requests, newest first, paginating
// within the lookback window.
func (p *gitlabProvider) list(ctx context.Context, cutoff time.Time) ([]pullRequest, error) {
	return paginatePRList(ctx, p, cutoff)
}

// fetchPRPage fetches one page of merge requests ordered by update time descending.
func (p *gitlabProvider) fetchPRPage(ctx context.Context, page int) ([]pullRequest, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/merge_requests?state=all&order_by=updated_at&sort=desc&scope=all&page=%d&per_page=%d",
		p.apiBase, gitlabEscapeProjectPath(p.projectPath), page, gitlabMRListPageSize)

	var mrs []gitlabMR
	if err := prAPIGet(ctx, "GitLab", endpoint, p.token, gitlabAcceptJSON, &mrs); err != nil {
		return nil, p.describeError(err)
	}

	prs := make([]pullRequest, 0, len(mrs))
	for _, mr := range mrs {
		prs = append(prs, pullRequest{
			Number:    mr.IID,
			State:     normalizeGitLabState(mr.State),
			Title:     mr.Title,
			HeadRef:   mr.SourceBranch,
			HeadSHA:   mr.SHA,
			BaseRef:   mr.TargetBranch,
			Author:    mr.Author.Username,
			URL:       mr.WebURL,
			UpdatedAt: mr.UpdatedAt,
			Draft:     mr.Draft,
		})
	}
	return prs, nil
}

// changedFiles returns the paths touched by a merge request. GitLab caps the diffs
// endpoint at a small page size, so results are paginated. truncated reports that
// the page cap was reached before the diff was exhausted, in which case the list is
// incomplete.
func (p *gitlabProvider) changedFiles(ctx context.Context, iid int) ([]string, bool, error) {
	paths := make([]string, 0, gitlabMRDiffPageSize)
	seen := make(map[string]struct{}, gitlabMRDiffPageSize)

	addPath := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}

	for page := 1; page <= gitlabMRDiffMaxPages; page++ {
		endpoint := fmt.Sprintf("%s/projects/%s/merge_requests/%d/diffs?page=%d&per_page=%d",
			p.apiBase, gitlabEscapeProjectPath(p.projectPath), iid, page, gitlabMRDiffPageSize)

		var diffs []gitlabMRDiff
		if err := prAPIGet(ctx, "GitLab", endpoint, p.token, gitlabAcceptJSON, &diffs); err != nil {
			return nil, false, p.describeError(err)
		}

		for _, diff := range diffs {
			addPath(diff.NewPath)
			// A rename moves a file between paths, so match on both. This keeps a
			// file that moved out of a watched directory visible to path filters.
			if diff.RenamedFile {
				addPath(diff.OldPath)
			}
		}

		if len(diffs) < gitlabMRDiffPageSize {
			return paths, false, nil
		}
	}

	log.DefaultLogger.Warnf("git informer: merge request !%d in %s has more changed files than the %d fetched; path filtering may be incomplete",
		iid, p.projectPath, gitlabMRDiffMaxPages*gitlabMRDiffPageSize)
	return paths, true, nil
}

// describeError adds actionable context to GitLab's deliberately vague 404, which
// it returns for both a wrong project path and a token without sufficient access.
func (p *gitlabProvider) describeError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "returned 404") {
		return fmt.Errorf("%w (check that the project path %q is correct, including any subgroups, and that the token has the read_api scope)",
			err, p.projectPath)
	}
	return err
}

// gitlabAPIBaseFromURI returns the GitLab REST API v4 base URL for a repo URI. A
// port is only carried over from http/https URIs, never from SSH.
func gitlabAPIBaseFromURI(uri string) string {
	_, authority := gitAPIHostPort(uri)
	if authority == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/api/v4", authority)
}

// gitlabEscapeProjectPath percent-encodes a namespaced project path for use as the
// :id path parameter, e.g. "group/sub/project" -> "group%2Fsub%2Fproject".
//
// The encoded form must survive all the way onto the wire: building the endpoint
// as a string and letting http.NewRequest parse it preserves the escaping in
// URL.RawPath, whereas assigning to url.URL.Path would re-encode the %2F back to
// literal slashes and GitLab would answer 404.
func gitlabEscapeProjectPath(projectPath string) string {
	return url.PathEscape(strings.Trim(projectPath, "/"))
}

// normalizeGitLabState maps GitLab merge request states onto the normalized
// open/closed vocabulary shared with GitHub.
//
// "locked" is a short-lived merge-in-progress state and maps to open, so that an
// opened -> locked -> merged sequence fires a single "closed" event rather than a
// spurious extra event on the locked hop. Unrecognized states are treated as open
// so a future GitLab state cannot fire a bogus "closed".
func normalizeGitLabState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case gitlabStateClosed, gitlabStateMerged:
		return prStateClosed
	case gitlabStateOpened, gitlabStateLocked:
		return prStateOpen
	default:
		return prStateOpen
	}
}
