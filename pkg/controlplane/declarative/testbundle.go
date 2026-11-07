package declarative

import (
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation/field"

	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
)

// MaxTestBundleNameLength bounds a Test Bundle's slug, which labels its
// workflow and sandbox resources.
const MaxTestBundleNameLength = 59

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

// validateGitURI refuses an empty repository URI and one that carries
// credentials: a password, any user of an http or https URL, where a token
// often goes, or a query parameter whose name IsCredentialField. scp-like and ssh URIs, such as git@github.com:org/repo,
// keep their user, which names an account rather than holding a secret. The
// URI is never echoed.
func validateGitURI(raw string, path *field.Path) field.ErrorList {
	if strings.TrimSpace(raw) == "" {
		return field.ErrorList{field.Required(path, "the repository to read tests from")}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		// An scp-like URI (user@host:path) is not a URL; it holds no password.
		return nil
	}
	refused := false
	if u.User != nil {
		_, hasPassword := u.User.Password()
		refused = hasPassword || slices.Contains([]string{"http", "https"}, strings.ToLower(u.Scheme))
	}
	for name := range u.Query() {
		refused = refused || IsCredentialField(name)
	}
	if refused {
		return field.ErrorList{invalidSecret(path, "must not carry credentials; connect the repository in the Control Plane instead")}
	}
	return nil
}
