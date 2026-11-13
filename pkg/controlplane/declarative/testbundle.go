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

	// A bundle's Skills are always included: every authoring prompt carries them.
	skills := make([]skillFields, 0, len(spec.Skills))
	total := 0
	for i := range spec.Skills {
		skill := &spec.Skills[i]
		skills = append(skills, skillFields{name: &skill.Name, description: &skill.Description, content: skill.Content, metadata: skill.Metadata})
		total += utf8.RuneCountInString(skill.Content)
	}
	errs = append(errs, normalizeSkills(skills, total, path.Child("skills"))...)
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

// validateGitURI refuses an empty repository URI and one that is neither a
// URL with a scheme and host nor scp-like. A URI may carry credentials, such
// as a token in an https URL, as the REST API accepts, so a refused one is
// never echoed.
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
	return nil
}
