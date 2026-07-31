package informer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

// providerKind identifies a git hosting provider for pull request polling.
type providerKind string

const (
	providerUnknown providerKind = ""
	providerGitHub  providerKind = "github"
	providerGitLab  providerKind = "gitlab"
)

const prGitHubAuthTypeWrongProviderWarning = "github authType is only supported for GitHub repositories, falling back to configured credentials"

// splitGitURI normalizes an HTTPS/HTTP/SSH/scp-style git URI into its bare host
// (no userinfo, no port) and its repository path (no leading or trailing slash,
// no ".git" suffix, no GitLab "/-/..." web UI suffix). ok is false when either
// component cannot be derived.
func splitGitURI(uri string) (host, repoPath string, ok bool) {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return "", "", false
	}

	var rawPath string
	if strings.Contains(trimmed, "://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", "", false
		}
		host = u.Hostname()
		rawPath = u.Path
	} else {
		// scp-like syntax: [user@]host:path
		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			return "", "", false
		}
		host = trimmed[:idx]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		rawPath = trimmed[idx+1:]
	}

	host = strings.ToLower(host)
	repoPath = normalizeRepoPath(rawPath)
	if host == "" || repoPath == "" {
		return "", "", false
	}
	return host, repoPath, true
}

// normalizeRepoPath trims a repository path down to its namespaced project path.
func normalizeRepoPath(rawPath string) string {
	path := strings.Trim(rawPath, "/")
	// Strip a GitLab web UI suffix such as "/-/merge_requests/7" so a URL copied
	// from the browser still resolves to the project.
	if idx := strings.Index(path, "/-/"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimSuffix(path, ".git")
	return strings.Trim(path, "/")
}

// gitAPIHostPort returns the bare host and the authority (host[:port]) to use for
// REST API calls. A port is only carried over from http/https URIs, because an
// SSH port must never be reused for HTTPS.
func gitAPIHostPort(uri string) (host, authority string) {
	trimmed := strings.TrimSpace(uri)
	if strings.Contains(trimmed, "://") {
		if u, err := url.Parse(trimmed); err == nil && u.Hostname() != "" {
			host = strings.ToLower(u.Hostname())
			scheme := strings.ToLower(u.Scheme)
			if scheme == "http" || scheme == "https" {
				return host, strings.ToLower(u.Host)
			}
			return host, host
		}
	}
	host, _, ok := splitGitURI(uri)
	if !ok {
		return "", ""
	}
	return host, host
}

// providerKindFromHost resolves a provider from the host alone, without network
// access. ok is false when the host is an unrecognized self-managed instance.
//
// Matching is on whole DNS labels, never on substrings: a substring test also
// accepts "notgithub.com" and "evil-gitlab.attacker.com", which have no legitimate
// use and only widen the set of hosts that can be handed a credential.
//
// No name test can make this a trust decision, though. An author who controls the
// trigger's uri can always name their host "gitlab.attacker.com", which is
// indistinguishable from the legitimate "gitlab.mycompany.com" convention. Gating
// credential release is the job of Options.AllowedPRHosts, not of this function.
func providerKindFromHost(host string) (providerKind, bool) {
	if host == "" {
		return providerUnknown, false
	}
	labels := strings.Split(host, ".")
	// GitLab is checked first so a host such as "gitlab.github-mirror.example.com"
	// is not misread as GitHub.
	for _, kind := range []providerKind{providerGitLab, providerGitHub} {
		name := string(kind)
		// The canonical host or a subdomain of it, e.g. gitlab.com, www.gitlab.com.
		if host == name+".com" || strings.HasSuffix(host, "."+name+".com") {
			return kind, true
		}
		// Self-managed convention: the leading label names the provider, e.g.
		// gitlab.example.com or github.corp.internal.
		if labels[0] == name {
			return kind, true
		}
	}
	return providerUnknown, false
}

// prProviderFor resolves which provider backs the trigger's repository and returns
// a client for it with the API base, repository identity and token resolved.
func (i *Informer) prProviderFor(
	ctx context.Context,
	namespace string,
	gitConfig *testkube.TestTriggerContentGit,
	cache *reconcileCache,
) (prProvider, error) {
	host, repoPath, ok := splitGitURI(gitConfig.Uri)
	if !ok {
		return nil, fmt.Errorf("git-pull-request trigger requires a repository URL with a host and project path, got: %s",
			sanitizeTokenURI(gitConfig.Uri))
	}

	if !i.prHostAllowed(host) {
		return nil, fmt.Errorf("git-pull-request trigger host %q is not in the configured pull request host allowlist (%s)",
			host, strings.Join(i.options.AllowedPRHosts, ", "))
	}

	kind, err := i.resolveProviderKind(ctx, namespace, host, gitConfig)
	if err != nil {
		return nil, err
	}

	token := i.resolvePRToken(ctx, namespace, gitConfig, kind, cache)

	switch kind {
	case providerGitLab:
		if !strings.Contains(repoPath, "/") {
			return nil, fmt.Errorf("git-pull-request trigger requires a namespaced GitLab project path (group/project), got: %s", repoPath)
		}
		return newGitLabProvider(i.prAPIBase(gitConfig.Uri, providerGitLab), repoPath, token), nil
	default:
		owner, repo, parsed := splitGitHubRepoPath(repoPath)
		if !parsed {
			return nil, fmt.Errorf("git-pull-request trigger requires a GitHub repository URL of the form host/owner/repo, got: %s",
				sanitizeTokenURI(gitConfig.Uri))
		}
		return newGitHubProvider(i.prAPIBase(gitConfig.Uri, providerGitHub), owner, repo, token), nil
	}
}

// splitGitHubRepoPath splits a normalized repository path into owner and repo.
// GitHub has no nested namespaces, so anything other than two segments is not a
// repository path.
//
// This is anchored on the path that splitGitURI derived from the URI, which is why
// it must not be replaced by a substring match on the whole URI: a pattern looking
// for "github.com" anywhere would accept
// https://evil.example.com/github.com/owner/repo and then send the trigger's token
// to evil.example.com, because the API base is derived from the host while the
// repository would have been derived from the path.
func splitGitHubRepoPath(repoPath string) (owner, repo string, ok bool) {
	segments := strings.Split(repoPath, "/")
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		return "", "", false
	}
	return segments[0], segments[1], true
}

// resolveProviderKind determines the provider for a host, consulting the process
// lifetime cache first and probing only for hosts whose name does not identify the
// provider.
func (i *Informer) resolveProviderKind(
	ctx context.Context,
	namespace, host string,
	gitConfig *testkube.TestTriggerContentGit,
) (providerKind, error) {
	if kind, ok := i.lookupPRProviderKind(host); ok {
		return kind, nil
	}

	if kind, ok := providerKindFromHost(host); ok {
		i.storePRProviderKind(host, kind)
		return kind, nil
	}

	// The GitHub App auth type can only ever mean GitHub, so an unrecognized host
	// using it needs no probe.
	if strings.EqualFold(gitConfig.AuthType, string(testkube.GITHUB_ContentGitAuthType)) {
		i.storePRProviderKind(host, providerGitHub)
		return providerGitHub, nil
	}

	kind, err := i.probeProviderKind(ctx, gitConfig)
	if err != nil {
		// Leave the host uncached so a transient failure does not pin it to the
		// wrong provider for the process lifetime.
		return providerUnknown, err
	}
	i.storePRProviderKind(host, kind)
	return kind, nil
}

// probeProviderKind detects an unrecognized self-managed host by asking it for the
// GitLab API version.
//
// The probe is deliberately UNAUTHENTICATED. The host here comes straight from the
// trigger's uri and has not yet been established as a supported provider, so it
// must never receive the trigger's credential: a 401 answer already proves the
// /api/v4 route exists, which is all the probe needs to decide.
func (i *Informer) probeProviderKind(
	ctx context.Context,
	gitConfig *testkube.TestTriggerContentGit,
) (providerKind, error) {
	uri := sanitizeTokenURI(gitConfig.Uri)
	endpoint := i.prAPIBase(gitConfig.Uri, providerGitLab) + "/version"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return providerUnknown, err
	}

	resp, err := prHTTPClient.Do(req)
	if err != nil {
		return providerUnknown, fmt.Errorf("failed to detect git provider for %s: %w", uri, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		log.DefaultLogger.Infof("git informer: detected GitLab API at %s", uri)
		return providerGitLab, nil
	case http.StatusNotFound:
		log.DefaultLogger.Infof("git informer: no GitLab API at %s, treating it as GitHub Enterprise Server", uri)
		return providerGitHub, nil
	default:
		return providerUnknown, fmt.Errorf("failed to detect git provider for %s: unexpected status %d from the GitLab version endpoint",
			uri, resp.StatusCode)
	}
}

// prAPIBase resolves the REST API base URL for the given repository and provider.
// Tests override prAPIBaseFunc to redirect calls at a mock server.
func (i *Informer) prAPIBase(uri string, kind providerKind) string {
	if i.prAPIBaseFunc != nil {
		return i.prAPIBaseFunc(uri)
	}
	if kind == providerGitLab {
		return gitlabAPIBaseFromURI(uri)
	}
	return githubAPIBaseFromURI(uri)
}

func (i *Informer) lookupPRProviderKind(host string) (providerKind, bool) {
	if i.prProviders == nil {
		return providerUnknown, false
	}
	kind, ok := i.prProviders[host]
	return kind, ok
}

func (i *Informer) storePRProviderKind(host string, kind providerKind) {
	if i.prProviders == nil {
		i.prProviders = make(map[string]providerKind)
	}
	i.prProviders[host] = kind
}

// prHostAllowed reports whether a repository host may be contacted with a trigger
// credential.
//
// This is the only real control over where credentials go. A trigger author chooses
// both the uri and the tokenFrom Secret reference, and tokenFrom may name any Secret
// in the trigger's namespace, so without an allowlist an author can have the
// informer send a Secret they cannot otherwise read to a host of their choosing.
// Host naming cannot substitute for this: "gitlab.attacker.com" satisfies every
// name-based test.
//
// The allowlist is opt-in. When empty every host is permitted, which preserves the
// behaviour of existing installations; operators who care set AllowedPRHosts.
// Entries match a host exactly or, when written as ".example.com", any subdomain.
func (i *Informer) prHostAllowed(host string) bool {
	if len(i.options.AllowedPRHosts) == 0 {
		return true
	}
	for _, allowed := range i.options.AllowedPRHosts {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if strings.HasPrefix(allowed, ".") {
			if host == strings.TrimPrefix(allowed, ".") || strings.HasSuffix(host, allowed) {
				return true
			}
			continue
		}
		if host == allowed {
			return true
		}
	}
	return false
}

// resolvePlainToken resolves the token configured directly on the trigger, without
// consulting the control plane's GitHub App integration.
func (i *Informer) resolvePlainToken(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) string {
	if i.kubeClient != nil {
		return i.resolveCredentialValue(ctx, gitConfig.Token, namespace, gitConfig.TokenFrom)
	}
	return resolveCredentialValue(gitConfig.Token, gitConfig.TokenFrom)
}
