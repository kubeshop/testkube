package declarative

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
)

// The bounds and patterns below mirror what the Control Plane accepts for an
// LLM provider, so a manifest it would refuse fails in the CLI first.
const (
	// MaxLLMProviderModels bounds the models of one provider.
	MaxLLMProviderModels = 50
	// MaxLLMModelNameLength bounds a model name.
	MaxLLMModelNameLength = 200
	// MaxHeaderEntries bounds the headers, and the query parameters, of a
	// provider or a model.
	MaxHeaderEntries = 50
	// RedactedValue is what the Control Plane exports in place of a header or
	// query parameter value. Applied back, it keeps the stored value.
	RedactedValue = "********"
)

// llmModelNamePattern keeps model names that LiteLLM takes literally: "*"
// would make a wildcard deployment and "," splits the Sandbox's model list.
var llmModelNamePattern = regexp.MustCompile(`^[A-Za-z0-9._:/@-]+$`)

// Credential reference schemes. A bare name is a credential too.
const (
	credentialScheme = "secret://"
	vaultScheme      = "vault://"
)

// forbiddenHeaders are headers the Control Plane never forwards to a provider:
// they would redirect the request or reach a cloud metadata service.
var forbiddenHeaders = []string{"host", "transfer-encoding", "content-length", "metadata-flavor", "x-aws-ec2-metadata-token", "x-goog-metadata-request", "x-gce-metadata-flavor", "x-azure-fed-token-request"}

// AgentRoles are the environment roles an AI Agent can be linked to.
var AgentRoles = []string{"admin", "read", "run", "write"}

// credentialNamePattern is the name of an Environment Credential.
var credentialNamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// ParseCredentialReference splits a credentialReference into the name it
// names and its source: "credential" for secret://<name> or a bare name, and
// "vault" for vault://<path>.
func ParseCredentialReference(ref string) (name, source string) {
	if after, ok := strings.CutPrefix(ref, credentialScheme); ok {
		return after, "credential"
	}
	if after, ok := strings.CutPrefix(ref, vaultScheme); ok {
		return after, "vault"
	}
	return ref, "credential"
}

// ProposeName returns a metadata.name for an AI resource that has only a
// display name, as one made before declarative AI resources existed: lower
// case, with every run of other characters turned into one '-', and cut to
// a DNS-1123 label. It returns "" when nothing is left; the caller then
// picks a name of its own, as it does to make a taken one unique.
func ProposeName(displayName string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(displayName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	name := strings.TrimSuffix(b.String(), "-")
	if len(name) > validation.DNS1123LabelMaxLength {
		name = strings.TrimRight(name[:validation.DNS1123LabelMaxLength], "-")
	}
	return name
}

// validateAIName checks the metadata.name of an AI resource: a DNS-1123
// label, so the same name works for a custom resource synced from a cluster.
func validateAIName(name string, path *field.Path) field.ErrorList {
	if name == "" {
		return field.ErrorList{field.Required(path, "the resource's name")}
	}
	var errs field.ErrorList
	for _, msg := range validation.IsDNS1123Label(name) {
		errs = append(errs, field.Invalid(path, name, msg))
	}
	return errs
}

func defaultDisplayName(displayName *string, name string) {
	*displayName = strings.TrimSpace(*displayName)
	if *displayName == "" {
		*displayName = name
	}
}

func normalizeMCPServer(s *aiv1.MCPServer) field.ErrorList {
	namePath := field.NewPath("metadata", "name")
	errs := validateAIName(s.Name, namePath)
	if s.Name == aiv1.BuiltinMCPServerName {
		errs = append(errs, field.Invalid(namePath, s.Name, "is the name of the built-in Testkube MCP server"))
	}
	spec := &s.Spec
	path := field.NewPath("spec")
	defaultDisplayName(&spec.DisplayName, s.Name)
	if spec.Type == "" {
		spec.Type = aiv1.MCPServerTypeHTTP
	}
	// The Control Plane takes a URL for every server, stdio too, as its REST
	// API does.
	if spec.URL == "" {
		errs = append(errs, field.Required(path.Child("url"), "the URL of the server's endpoint"))
	} else {
		errs = append(errs, validateHTTPURL(spec.URL, path.Child("url"))...)
	}
	switch spec.Type {
	case aiv1.MCPServerTypeHTTP, aiv1.MCPServerTypeSSE:
		if spec.Command != "" {
			errs = append(errs, field.Forbidden(path.Child("command"), "only a stdio server runs a command"))
		}
		if len(spec.Args) > 0 {
			errs = append(errs, field.Forbidden(path.Child("args"), "only a stdio server runs a command"))
		}
	case aiv1.MCPServerTypeStdio:
		if spec.Command == "" {
			errs = append(errs, field.Required(path.Child("command"), "a stdio server needs a command to run"))
		}
	default:
		errs = append(errs, field.NotSupported(path.Child("type"), spec.Type, []string{aiv1.MCPServerTypeHTTP, aiv1.MCPServerTypeSSE, aiv1.MCPServerTypeStdio}))
	}
	return errs
}

func normalizeLLMProvider(p *aiv1.LLMProvider) field.ErrorList {
	errs := validateAIName(p.Name, field.NewPath("metadata", "name"))
	spec := &p.Spec
	path := field.NewPath("spec")
	defaultDisplayName(&spec.DisplayName, p.Name)
	spec.Provider = strings.TrimSpace(spec.Provider)
	if spec.Provider == "" {
		errs = append(errs, field.Required(path.Child("provider"), "the kind of provider, such as openai"))
	}
	if spec.BaseURL != "" {
		errs = append(errs, validateHTTPURL(spec.BaseURL, path.Child("baseUrl"))...)
	}
	if spec.CredentialReference != "" {
		errs = append(errs, validateCredentialReference(spec.CredentialReference, true, path.Child("credentialReference"))...)
	}
	errs = append(errs, validateForwarded(spec.ExtraHeaders, true, path.Child("extraHeaders"))...)
	errs = append(errs, validateForwarded(spec.QueryParams, false, path.Child("queryParams"))...)

	modelsPath := path.Child("models")
	if len(spec.Models) > MaxLLMProviderModels {
		errs = append(errs, field.TooMany(modelsPath, len(spec.Models), MaxLLMProviderModels))
	}
	seen := map[string]bool{}
	for i := range spec.Models {
		m := &spec.Models[i]
		mPath := modelsPath.Index(i)
		m.Name = strings.TrimSpace(m.Name)
		switch {
		case m.Name == "":
			errs = append(errs, field.Required(mPath.Child("name"), "the model's name at the provider"))
		case len(m.Name) > MaxLLMModelNameLength:
			errs = append(errs, field.TooLong(mPath.Child("name"), m.Name, MaxLLMModelNameLength))
		case !llmModelNamePattern.MatchString(m.Name):
			errs = append(errs, field.Invalid(mPath.Child("name"), m.Name, "may contain only letters, digits and . _ : / @ -"))
		case seen[m.Name]:
			errs = append(errs, field.Duplicate(mPath.Child("name"), m.Name))
		}
		seen[m.Name] = true
		if m.URL != "" {
			errs = append(errs, validateHTTPURL(m.URL, mPath.Child("url"))...)
		}
		if m.ContextWindowTokens < 0 {
			errs = append(errs, field.Invalid(mPath.Child("contextWindowTokens"), m.ContextWindowTokens, "must not be negative"))
		}
		errs = append(errs, validateForwarded(m.ExtraHeaders, true, mPath.Child("extraHeaders"))...)
		errs = append(errs, validateForwarded(m.QueryParams, false, mPath.Child("queryParams"))...)
	}
	return errs
}

func normalizeAIAgent(a *aiv1.AIAgent) field.ErrorList {
	errs := validateAIName(a.Name, field.NewPath("metadata", "name"))
	spec := &a.Spec
	path := field.NewPath("spec")
	defaultDisplayName(&spec.DisplayName, a.Name)
	if strings.TrimSpace(spec.Instructions) == "" {
		errs = append(errs, field.Required(path.Child("instructions"), "the agent's system prompt"))
	}
	for key := range spec.Labels {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, field.Invalid(path.Child("labels"), key, "a label needs a key"))
		}
	}

	serversPath := path.Child("mcpServers")
	seen := map[string]bool{}
	for i := range spec.MCPServers {
		link := &spec.MCPServers[i]
		linkPath := serversPath.Index(i)
		switch {
		case link.Name == "":
			errs = append(errs, field.Required(linkPath.Child("name"), "the MCPServer's name, or testkube for the built-in server"))
		case seen[link.Name]:
			errs = append(errs, field.Duplicate(linkPath.Child("name"), link.Name))
		case link.Name != aiv1.BuiltinMCPServerName:
			errs = append(errs, validateAIName(link.Name, linkPath.Child("name"))...)
		}
		seen[link.Name] = true
		switch link.SignIn {
		case "", aiv1.SignInNone, aiv1.SignInSharedKey, aiv1.SignInPerPerson, aiv1.SignInOBO:
		default:
			errs = append(errs, field.NotSupported(linkPath.Child("signIn"), link.SignIn, []string{aiv1.SignInNone, aiv1.SignInSharedKey, aiv1.SignInPerPerson, aiv1.SignInOBO}))
		}
		if link.CredentialReference != "" {
			if link.SignIn != aiv1.SignInSharedKey {
				errs = append(errs, field.Forbidden(linkPath.Child("credentialReference"), "only a shared_key link signs in with a credential"))
			} else {
				errs = append(errs, validateCredentialReference(link.CredentialReference, false, linkPath.Child("credentialReference"))...)
			}
		}
		for tool := range link.Tools {
			if strings.TrimSpace(tool) == "" {
				errs = append(errs, field.Invalid(linkPath.Child("tools"), tool, "a tool needs a name"))
			}
		}
	}

	if spec.Access != nil {
		accessPath := path.Child("access")
		rolesPath := accessPath.Child("roles")
		errs = append(errs, normalizeNames(&spec.Access.Roles, rolesPath)...)
		for i, role := range spec.Access.Roles {
			if role != "" && !slices.Contains(AgentRoles, role) {
				errs = append(errs, field.NotSupported(rolesPath.Index(i), role, AgentRoles))
			}
		}
		errs = append(errs, normalizeNames(&spec.Access.ResourceGroups, accessPath.Child("resourceGroups"))...)
	}

	skills := make([]skillFields, 0, len(spec.Skills))
	alwaysIncluded := 0
	for i := range spec.Skills {
		s := &spec.Skills[i]
		skills = append(skills, skillFields{name: &s.Name, description: &s.Description, content: s.Content, metadata: s.Metadata})
		if s.AlwaysInclude {
			alwaysIncluded += utf8.RuneCountInString(s.Content)
		}
	}
	errs = append(errs, normalizeSkills(skills, alwaysIncluded, path.Child("skills"))...)
	return errs
}

// normalizeNames sorts a list of names, as the Control Plane returns them, so
// an export applies without a diff, and refuses an empty or repeated one.
func normalizeNames(names *[]string, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	seen := map[string]bool{}
	for i, name := range *names {
		switch {
		case strings.TrimSpace(name) == "":
			errs = append(errs, field.Required(path.Index(i), "drop empty entries"))
		case seen[name]:
			errs = append(errs, field.Duplicate(path.Index(i), name))
		}
		seen[name] = true
	}
	slices.Sort(*names)
	return errs
}

// invalidSecret is an Invalid error that leaves the value out. Errors reach
// the CLI's stderr, and from there terminal and CI logs, so a value refused
// for possibly being, or holding, a secret is never echoed: a URL, a
// credentialReference.
func invalidSecret(path *field.Path, detail string) *field.Error {
	return field.Invalid(path, field.OmitValueType{}, detail)
}

// validateHTTPURL checks an endpoint is an absolute http or https URL. It may
// carry credentials, as the REST API accepts, so a refused URL is not echoed.
func validateHTTPURL(raw string, path *field.Path) field.ErrorList {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return field.ErrorList{invalidSecret(path, "must be an absolute http or https URL")}
	}
	return nil
}

// validateCredentialReference checks a reference names a credential rather
// than holding one. allowVault admits vault://<path>, which only LLM providers
// read. A refused reference may be the key itself, so it is never echoed.
func validateCredentialReference(ref string, allowVault bool, path *field.Path) field.ErrorList {
	name, source := ParseCredentialReference(ref)
	if source == "vault" {
		if !allowVault {
			return field.ErrorList{invalidSecret(path, "must name an Environment Credential, as secret://<name>")}
		}
		if name == "" || strings.ContainsAny(name, " \t\n") {
			return field.ErrorList{invalidSecret(path, "must be vault://<path>")}
		}
		return nil
	}
	if !credentialNamePattern.MatchString(name) {
		return field.ErrorList{invalidSecret(path, "must name an Environment Credential, as secret://<name>; a key itself never goes in a manifest")}
	}
	return nil
}

// validateForwarded checks the headers or query parameters sent to a
// provider: at most MaxHeaderEntries, and for headers none the Control Plane
// refuses to forward, as the REST API does. A value may be a credential, as
// an X-Api-Key header is; it is never echoed in an error.
//
// The Control Plane exports every value as RedactedValue, and keeps the stored
// value for one applied back that way, as its REST API does. (It refuses the
// placeholder for a value it does not have.)
func validateForwarded(values map[string]string, headers bool, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if len(values) > MaxHeaderEntries {
		errs = append(errs, field.TooMany(path, len(values), MaxHeaderEntries))
	}
	if !headers {
		return errs
	}
	for _, name := range sortedKeys(values) {
		if slices.Contains(forbiddenHeaders, strings.ToLower(name)) {
			errs = append(errs, field.Forbidden(path.Key(name), "is a header the Control Plane does not forward"))
		}
	}
	return errs
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// unsupportedType is the error for an object of no registered kind.
func unsupportedType(obj Object) error {
	return fmt.Errorf("unsupported object type %T", obj)
}

// ValidateAIName returns why a slug given to an AI Agent, MCP server or LLM
// provider through another path than a manifest, such as the REST API, is
// not one a manifest could name it by, or nil.
func ValidateAIName(name string) error {
	return validateAIName(name, field.NewPath("slug")).ToAggregate()
}
