// Package informer polls git content selected by TestTriggers and fires a match when it changes.
// Only the elected leader runs it (the leader coordinator tasks in cmd/api-server/main.go).
//
// # Push and tag triggers
//
// git-push and git-tag-push clone or fetch the repository with go-git and compare commits.
// Branch and tag filters, path filters, and credential resolution live in informer.go.
//
// # Pull request triggers
//
// A git-pull-request trigger (event or events; consumers use EffectiveEvents) does not clone.
// It polls the provider REST API. The provider is detected from the repository URI:
// github.com and a host whose first label is github are GitHub; gitlab.com and a host whose
// first label is gitlab are GitLab. Any other self-managed host is probed, unauthenticated,
// at <api>/version. A 200, 401, or 403 means GitLab; a 404 is treated as GitHub Enterprise
// Server. The result is cached for the process lifetime per API authority, which is the host
// plus the HTTP or HTTPS port. Two ports on one host are cached separately, because one port
// can be GitLab and another GitHub Enterprise Server. An SSH port is not part of the key:
// REST calls never reuse it.
//
// GitHub pull requests and GitLab merge requests share one action vocabulary: opened,
// synchronize, reopened, and closed. A merged GitLab merge request reports closed. GitHub App
// auth (authType github) is used only for GitHub; GitLab uses the token or tokenFrom on the
// trigger, which needs the read_api scope.
//
// # Host allowlist
//
// The URI and tokenFrom are both chosen by the trigger author, and tokenFrom may name any
// Secret in the trigger's namespace. Options.AllowedGitHosts is the only gate on where that
// credential is sent. An empty list means the built-in default: github.com, gitlab.com, and
// their subdomains. The API server does not leave the list empty. It sets
// TEST_TRIGGER_GIT_INFORMER_ALLOWED_HOSTS to "*" so an upgrade does not break self-managed
// hosts, and the informer logs a warning when it starts with that wildcard. An operator
// restricts the list to their own hosts in production. A single "*" entry allows any host.
// An entry with a leading dot matches that domain and its subdomains.
package informer
