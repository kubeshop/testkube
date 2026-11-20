package declarative

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation/field"

	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
)

// MaxTestBundleNameLength bounds a Test Bundle's slug, which labels its
// workflow and sandbox resources.
const MaxTestBundleNameLength = 59

// scpGitURIPattern matches Git's scp-like syntax, [user@]host:path, as Git
// tells it from a URL: no "://", and no slash before the colon. It captures
// the user and the path, which may hold an '@', as team/repo@v2.git.
var scpGitURIPattern = regexp.MustCompile(`^(?:([^@/:]+)@)?(?:[^@/:\[\]]+|\[[^\]/]+\]):(.*)$`)

// isSCPGitURI reports whether raw is a scp-like address. One with no user
// whose path has an '@' before its first '/' is not: it reads as a URL with
// its "//" left out, as https:token@github.com/org/repo, rather than as a
// repository on the host "https". So an '@' is accepted in the path only
// after a user, or after the path's first '/'.
func isSCPGitURI(raw string) bool {
	if strings.Contains(raw, "://") {
		return false
	}
	m := scpGitURIPattern.FindStringSubmatch(raw)
	if m == nil {
		return false
	}
	user, path := m[1], m[2]
	firstSegment, _, _ := strings.Cut(path, "/")
	return user != "" || !strings.Contains(firstSegment, "@")
}

func normalizeTestBundle(b *testbundlesv1.TestBundle) field.ErrorList {
	errs := validateTestBundleName(b.Name, field.NewPath("metadata", "name"))
	spec := &b.Spec
	path := field.NewPath("spec")
	defaultDisplayName(&spec.DisplayName, b.Name)
	for key := range spec.Labels {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, field.Invalid(path.Child("labels"), key, "a label needs a key"))
		}
	}
	if spec.GitSource != nil {
		errs = append(errs, validateGitURI(spec.GitSource.URI, path.Child("gitSource", "uri"))...)
	}

	skills := make([]skillFields, 0, len(spec.Skills))
	alwaysIncluded := 0
	for i := range spec.Skills {
		skill := &spec.Skills[i]
		skills = append(skills, skillFields{name: &skill.Name, description: &skill.Description, content: skill.Content, metadata: skill.Metadata})
		if skill.AlwaysInclude {
			alwaysIncluded += utf8.RuneCountInString(skill.Content)
		}
	}
	errs = append(errs, normalizeSkills(skills, alwaysIncluded, path.Child("skills"))...)
	errs = append(errs, normalizeTestBundleContexts(spec.Contexts, path.Child("contexts"))...)

	if spec.AIAgents != nil {
		// Checked before normalizeNames sorts them, so an error names the
		// entry as the manifest has it.
		agentsPath := path.Child("aiAgents")
		for i, name := range *spec.AIAgents {
			if strings.TrimSpace(name) != "" {
				errs = append(errs, validateAIName(name, agentsPath.Index(i))...)
			}
		}
		errs = append(errs, normalizeNames(spec.AIAgents, agentsPath)...)
	}
	return errs
}

// normalizeTestBundleContexts trims each Attached Context's title and checks
// that it has a title and content, as the dashboard does. Titles may repeat,
// as they may there, so every bundle's Attached Context can be exported and
// applied back. No content is echoed.
func normalizeTestBundleContexts(contexts []testbundlesv1.TestBundleContext, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	for i := range contexts {
		c := &contexts[i]
		c.Title = strings.TrimSpace(c.Title)
		if c.Title == "" {
			errs = append(errs, field.Required(path.Index(i).Child("title"), "a title such as Checkout API conventions"))
		}
		if strings.TrimSpace(c.Content) == "" {
			errs = append(errs, field.Required(path.Index(i).Child("content"), "the guidance the AI reads"))
		}
	}
	return errs
}

// validateTestBundleName checks the metadata.name of a Test Bundle: the slug
// the Control Plane gives a bundle, so every existing bundle can be named.
// The slug of a bundle made in the dashboard may hold '_', so unlike an AI
// resource's name it is not always a DNS-1123 label.
func validateTestBundleName(name string, path *field.Path) field.ErrorList {
	errs := validateSlug(name, path)
	if len(name) > MaxTestBundleNameLength {
		errs = append(errs, field.TooLong(path, "", MaxTestBundleNameLength))
	}
	return errs
}

// validateGitURI refuses an empty repository URI, one that is neither a URL
// with a scheme and host nor scp-like, and a URL that carries credentials
// (gitURLHasCredentials). A manifest is kept in Git and printed back by get,
// so a token in it would be readable by anyone who reads either. A refused
// URI is never echoed, since it may hold a token.
func validateGitURI(raw string, path *field.Path) field.ErrorList {
	if strings.TrimSpace(raw) == "" {
		return field.ErrorList{field.Required(path, "the repository to read tests from")}
	}
	if isSCPGitURI(raw) {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Opaque != "" || u.Host == "" {
		return field.ErrorList{invalidSecret(path, "must be a URL, such as https://github.com/org/repo.git, or an scp-like address, such as git@github.com:org/repo.git")}
	}
	if gitURLHasCredentials(u) {
		return field.ErrorList{invalidSecret(path, "must not carry credentials; name the repository without a user or password, such as https://github.com/org/repo.git")}
	}
	return nil
}

// gitURLHasCredentials reports whether a repository URL holds a password, or
// names a user over http or https, where the user is usually a token
// (https://ghp_...@github.com). A user alone over ssh, as in
// ssh://git@github.com, is not a secret.
func gitURLHasCredentials(u *url.URL) bool {
	if u.User == nil {
		return false
	}
	if _, ok := u.User.Password(); ok {
		return true
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.User.Username() != ""
	}
	return false
}

// GitURIWithoutCredentials is raw with the credentials gitURLHasCredentials
// finds left out, so the Control Plane can export a Git source URI that was
// stored with them (the dashboard and the REST API accept them) without
// printing a token, and the export applies back. It edits the string rather
// than parsing it, so the rest stays as written and a URL url.Parse cannot
// read loses its credentials too. A URI without them, and a scp-like one,
// whose user is no secret, are returned as they are.
func GitURIWithoutCredentials(raw string) string {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return raw
	}
	authority, tail := rest, ""
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		authority, tail = rest[:i], rest[i:]
	}
	// As url.Parse, the user information ends at the authority's last '@'.
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return raw
	}
	userinfo, host := authority[:at], authority[at+1:]
	switch strings.ToLower(scheme) {
	case "http", "https":
		return scheme + "://" + host + tail
	}
	// Only the password is a credential; ssh://git@host keeps its user.
	user, _, hasPassword := strings.Cut(userinfo, ":")
	if !hasPassword {
		return raw
	}
	return scheme + "://" + user + "@" + host + tail
}
