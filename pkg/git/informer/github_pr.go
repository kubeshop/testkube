package informer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// githubAcceptJSON is the recommended Accept header for the GitHub REST API.
	githubAcceptJSON = "application/vnd.github+json"

	// githubPRListPageSize is GitHub's documented maximum page size. Pages are
	// walked by paginatePRList within the lookback window.
	githubPRListPageSize = 100

	// githubPRFilesPageSize is GitHub's maximum page size. Pull requests with more
	// changed files than this will have incomplete path matching.
	githubPRFilesPageSize = 100
)

const githubPRNoTokenProviderWarning = "github authType configured for PR polling but no GitHub token provider is available, falling back to configured credentials"

// githubPR represents a minimal GitHub pull request from the REST API.
type githubPR struct {
	Number    int       `json:"number"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Head      struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	Draft bool `json:"draft"`
}

// githubPRFile represents a file changed in a pull request.
type githubPRFile struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename"`
	Status           string `json:"status"`
}

// githubAPIBaseFromURI returns the GitHub API base URL for the given repo URI.
// For github.com it returns "https://api.github.com", for GHES it derives from the
// host. A port is only carried over from http/https URIs, never from SSH.
func githubAPIBaseFromURI(uri string) string {
	host, authority := gitAPIHostPort(uri)
	if host == "" || host == "github.com" || strings.HasSuffix(host, "github.com") {
		return "https://api.github.com"
	}
	return fmt.Sprintf("https://%s/api/v3", authority)
}

// githubProvider polls the GitHub REST API for pull requests.
type githubProvider struct {
	apiBase string
	owner   string
	repo    string
	token   string
}

func newGitHubProvider(apiBase, owner, repo, token string) *githubProvider {
	return &githubProvider{apiBase: apiBase, owner: owner, repo: repo, token: token}
}

func (p *githubProvider) kind() providerKind { return providerGitHub }

func (p *githubProvider) headRef(number int) string {
	return "refs/pull/" + strconv.Itoa(number) + "/head"
}

func (p *githubProvider) listPageSize() int { return githubPRListPageSize }

// list fetches the most recently updated pull requests, newest first, paginating
// within the lookback window.
//
// GitHub's pulls endpoint has no update-time filter, so sort=updated&direction=desc
// plus a client-side cutoff is the only way to bound the walk.
func (p *githubProvider) list(ctx context.Context, cutoff time.Time) ([]pullRequest, error) {
	return paginatePRList(ctx, p, cutoff)
}

// fetchPRPage fetches one page of pull requests ordered by update time descending.
func (p *githubProvider) fetchPRPage(ctx context.Context, page int) ([]pullRequest, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all&sort=updated&direction=desc&page=%d&per_page=%d",
		p.apiBase, p.owner, p.repo, page, githubPRListPageSize)

	var githubPRs []githubPR
	if err := prAPIGet(ctx, "GitHub", endpoint, p.token, githubAcceptJSON, &githubPRs); err != nil {
		return nil, err
	}

	prs := make([]pullRequest, 0, len(githubPRs))
	for _, pr := range githubPRs {
		prs = append(prs, pullRequest{
			Number: pr.Number,
			// GitHub already reports open/closed, matching the normalized states.
			State:   pr.State,
			Title:   pr.Title,
			HeadRef: pr.Head.Ref,
			HeadSHA: pr.Head.SHA,
			BaseRef: pr.Base.Ref,
			Author:  pr.User.Login,
			URL:     pr.HTMLURL,
			// Required: paginatePRList stops on this, so leaving it zero would
			// silently cap the walk at one page.
			UpdatedAt: pr.UpdatedAt,
			Draft:     pr.Draft,
		})
	}
	return prs, nil
}

// changedFiles returns the paths touched by a pull request. truncated reports that
// the pull request has at least a full page of changed files, in which case the
// list may be incomplete.
func (p *githubProvider) changedFiles(ctx context.Context, number int) ([]string, bool, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=%d",
		p.apiBase, p.owner, p.repo, number, githubPRFilesPageSize)

	var files []githubPRFile
	if err := prAPIGet(ctx, "GitHub", endpoint, p.token, githubAcceptJSON, &files); err != nil {
		return nil, false, err
	}

	// A full page means there may be more; GitHub does not say either way.
	truncated := len(files) >= githubPRFilesPageSize

	paths := make([]string, 0, len(files))
	for _, f := range files {
		if f.Filename != "" {
			paths = append(paths, f.Filename)
		}
		// A rename moves a file between paths, so match on both. This keeps a file
		// that moved out of a watched directory visible to path filters.
		if f.PreviousFilename != "" && f.PreviousFilename != f.Filename {
			paths = append(paths, f.PreviousFilename)
		}
	}
	return paths, truncated, nil
}
