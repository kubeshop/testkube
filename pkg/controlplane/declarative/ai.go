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

// ResourceGroupLabel puts a Test Workflow in a resource group. The AI kinds
// refuse it rather than drop it: an AI Agent joins resource groups through
// spec.access.resourceGroups, and the other AI kinds belong to none.
const ResourceGroupLabel = "testkube.io/group"

// forbiddenHeaders are headers the Control Plane never sends to a server, as a
// provider's extra headers or an AI Agent's sign-in headers: they would
// redirect the request or reach a cloud metadata service.
var forbiddenHeaders = []string{"host", "transfer-encoding", "content-length", "metadata-flavor", "x-aws-ec2-metadata-token", "x-goog-metadata-request", "x-gce-metadata-flavor", "x-azure-fed-token-request"}

// authorizationHeader is never an LLM provider's extra header: the Control
// Plane sends the provider's key as Authorization itself.
const authorizationHeader = "authorization"

// credentialExpressionPattern is one {{ credential("name") }} in a value, the
// way a Test Workflow or a webhook reads an Environment Credential.
var credentialExpressionPattern = regexp.MustCompile(`\{\{\s*credential\(\s*"([^"]*)"\s*\)\s*\}\}`)

// CredentialExpression is the value that reads the named Environment
// Credential, as an export writes it.
func CredentialExpression(name string) string {
	return `{{ credential("` + name + `") }}`
}

// CredentialNames returns the Environment Credentials a value reads, in order,
// one for each {{ credential("name") }} in it. ok is false when the value holds
// any other expression, or names a credential no credential can be named.
func CredentialNames(value string) (names []string, ok bool) {
	for _, match := range credentialExpressionPattern.FindAllStringSubmatch(value, -1) {
		if !credentialNamePattern.MatchString(match[1]) {
			return nil, false
		}
		names = append(names, match[1])
	}
	rest := credentialExpressionPattern.ReplaceAllString(value, "")
	if strings.Contains(rest, "{{") || strings.Contains(rest, "}}") {
		return nil, false
	}
	return names, true
}

// ExpandCredentials returns value with each {{ credential("name") }} replaced
// by what resolve returns for the name. The Control Plane reads an AI Agent's
// sign-in headers this way when the agent signs in.
func ExpandCredentials(value string, resolve func(name string) (string, error)) (string, error) {
	if _, ok := CredentialNames(value); !ok {
		return "", fmt.Errorf("a value may read a credential only as %s", CredentialExpression("name"))
	}
	var resolveErr error
	expanded := credentialExpressionPattern.ReplaceAllStringFunc(value, func(expr string) string {
		if resolveErr != nil {
			return ""
		}
		resolved, err := resolve(credentialExpressionPattern.FindStringSubmatch(expr)[1])
		if err != nil {
			resolveErr = err
			return ""
		}
		return resolved
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return expanded, nil
}

// APIKeyCredential returns the Environment Credential an LLM provider's apiKey
// reads. ok is false unless the value is a single {{ credential("name") }}.
func APIKeyCredential(apiKey string) (name string, ok bool) {
	names, ok := CredentialNames(apiKey)
	if !ok || len(names) != 1 || strings.TrimSpace(apiKey) != strings.TrimSpace(credentialExpressionPattern.FindString(apiKey)) {
		return "", false
	}
	return names[0], true
}

// AgentRoles are the environment roles an AI Agent can be linked to.
var AgentRoles = []string{"admin", "read", "run", "write"}

// credentialNamePattern is the name of an Environment Credential.
var credentialNamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// credentialFieldWords mark a header or query parameter whose value is a
// credential, by a word in its name. Such a value belongs in an Environment
// Credential, which the manifest names, not in the manifest.
var credentialFieldWords = []string{"auth", "token", "secret", "password", "passwd", "cookie", "apikey", "api-key", "api_key", "signature"}

// credentialFieldNames mark a header or query parameter whose value is a
// credential, by its whole name or its last part (after '-', '_' or '.'):
// too short to look for inside a name. sig is the signature of an Azure SAS
// URL and of other signed endpoints.
var credentialFieldNames = []string{"key", "sig"}

// IsCredentialField reports whether a header or query parameter name looks
// like it carries a credential: Authorization, Cookie, X-Api-Key, a name
// with token, secret or password in it, or one that is or ends in key or sig.
// A name holding a whole pasted entry ("key=sk-live-123", "X-Api-Key: ...")
// is judged by the part before its first '=' or ':' too.
func IsCredentialField(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if i := strings.IndexAny(n, "=:"); i >= 0 && isCredentialName(strings.TrimSpace(n[:i])) {
		return true
	}
	return isCredentialName(n)
}

func isCredentialName(n string) bool {
	for _, word := range credentialFieldWords {
		if strings.Contains(n, word) {
			return true
		}
	}
	for _, word := range credentialFieldNames {
		if n == word || strings.HasSuffix(n, "-"+word) || strings.HasSuffix(n, "_"+word) || strings.HasSuffix(n, "."+word) {
			return true
		}
	}
	return false
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

// refuseGroupLabel refuses the resource group label of a Test Workflow on an
// AI kind, which would otherwise be dropped without a word. hint says what to
// do instead.
func refuseGroupLabel(labels map[string]string, hint string) field.ErrorList {
	if _, ok := labels[ResourceGroupLabel]; !ok {
		return nil
	}
	return field.ErrorList{field.Forbidden(field.NewPath("metadata", "labels").Key(ResourceGroupLabel), hint)}
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
	errs = append(errs, refuseGroupLabel(s.Labels, "MCP servers don't belong to resource groups; link the AI Agents that use the server to the group instead")...)
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
	errs = append(errs, refuseGroupLabel(p.Labels, "LLM providers don't belong to resource groups")...)
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
	if spec.APIKey != "" {
		if _, ok := APIKeyCredential(spec.APIKey); !ok {
			errs = append(errs, invalidSecret(path.Child("apiKey"), fmt.Sprintf("must read an Environment Credential, as %s; a key itself never goes in a manifest", CredentialExpression("name"))))
		}
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
	errs = append(errs, refuseGroupLabel(a.Labels, "an AI Agent joins resource groups through spec.access.resourceGroups, as [{name: <group>}]")...)
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
		if len(link.Headers) > 0 {
			if link.SignIn != aiv1.SignInSharedKey {
				errs = append(errs, field.Forbidden(linkPath.Child("headers"), "only a shared_key link signs in with headers"))
			} else {
				errs = append(errs, ValidateSignInHeaders(link.Headers, linkPath.Child("headers"))...)
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
		errs = append(errs, normalizeResourceGroups(spec.Access.ResourceGroups, accessPath.Child("resourceGroups"))...)
	}

	skills := make([]skillFields, 0, len(spec.Skills))
	alwaysIncluded := 0
	for i := range spec.Skills {
		s := &spec.Skills[i]
		skills = append(skills, skillFields{name: &s.Name, description: &s.Description, content: s.Content, metadata: s.Metadata, from: s.SkillFrom})
		if s.AlwaysInclude {
			alwaysIncluded += utf8.RuneCountInString(s.Content)
		}
	}
	errs = append(errs, normalizeSkills(skills, alwaysIncluded, path.Child("skills"))...)
	ownSkills := map[string]bool{}
	for _, s := range spec.Skills {
		ownSkills[s.Name] = true
	}
	errs = append(errs, normalizeSkillRefs(spec.SkillRefs, ownSkills, path.Child("skillRefs"))...)
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
// for possibly being, or holding, a secret is never echoed: a URL, an apiKey,
// a sign-in header.
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

// normalizeResourceGroups sorts an AI Agent's resource groups by name, as the
// Control Plane returns them, and refuses an empty or repeated one.
func normalizeResourceGroups(groups []aiv1.AIAgentResourceGroup, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	seen := map[string]bool{}
	for i, group := range groups {
		namePath := path.Index(i).Child("name")
		switch {
		case strings.TrimSpace(group.Name) == "":
			errs = append(errs, field.Required(namePath, "the resource group's slug"))
		case seen[group.Name]:
			errs = append(errs, field.Duplicate(namePath, group.Name))
		}
		seen[group.Name] = true
	}
	slices.SortStableFunc(groups, func(a, b aiv1.AIAgentResourceGroup) int { return strings.Compare(a.Name, b.Name) })
	return errs
}

// ValidateSignInHeaders checks the headers an AI Agent signs in to an MCP
// server with. They are stored on the agent and exported as written, so no
// value may be a key: a value reads an Environment Credential only as
// {{ credential("name") }}, and a header that carries a credential has to read
// one. The Control Plane applies the same rules to headers sent to its REST
// API. A refused value may be the key itself, so it is never echoed.
func ValidateSignInHeaders(headers map[string]string, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if len(headers) > MaxHeaderEntries {
		errs = append(errs, field.TooMany(path, len(headers), MaxHeaderEntries))
	}
	for _, name := range sortedKeys(headers) {
		namePath := path.Key(name)
		names, ok := CredentialNames(headers[name])
		switch {
		case strings.TrimSpace(name) == "":
			errs = append(errs, field.Required(namePath, "a header needs a name"))
		case slices.Contains(forbiddenHeaders, strings.ToLower(name)):
			errs = append(errs, field.Forbidden(namePath, "is a header the Control Plane does not send"))
		case !ok:
			errs = append(errs, invalidSecret(namePath, fmt.Sprintf("may read a credential only as %s", CredentialExpression("name"))))
		case len(names) == 0 && IsCredentialField(name):
			errs = append(errs, invalidSecret(namePath, fmt.Sprintf("carries a credential, so it has to read one, as 'Bearer %s'; a key itself never goes in a manifest", CredentialExpression("name"))))
		}
	}
	return errs
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
		switch lower := strings.ToLower(name); {
		case slices.Contains(forbiddenHeaders, lower):
			errs = append(errs, field.Forbidden(path.Key(name), "is a header the Control Plane does not forward"))
		case lower == authorizationHeader:
			errs = append(errs, field.Forbidden(path.Key(name), "is set from apiKey; the Control Plane does not take it from an extra header"))
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
