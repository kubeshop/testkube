package declarative

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
)

func TestNormalize_AIDefaults(t *testing.T) {
	server := &aiv1.MCPServer{Spec: aiv1.MCPServerSpec{URL: "https://example.com/mcp"}}
	server.Name = "github"
	require.NoError(t, Normalize(server))
	assert.Equal(t, "github", server.Spec.DisplayName)
	assert.Equal(t, aiv1.MCPServerTypeHTTP, server.Spec.Type)

	agent := &aiv1.AIAgent{Spec: aiv1.AIAgentSpec{
		DisplayName:  "  Triage  ",
		Instructions: "Triage.",
		Access:       &aiv1.AIAgentAccess{Roles: []string{"write", "read"}, ResourceGroups: []aiv1.AIAgentResourceGroup{{Name: "b"}, {Name: "a"}}},
	}}
	agent.Name = "triage"
	require.NoError(t, Normalize(agent))
	assert.Equal(t, "Triage", agent.Spec.DisplayName)
	assert.Equal(t, []string{"read", "write"}, agent.Spec.Access.Roles, "sorted as the Control Plane returns them")
	assert.Equal(t, []aiv1.AIAgentResourceGroup{{Name: "a"}, {Name: "b"}}, agent.Spec.Access.ResourceGroups)

	agent.Spec.Access.ResourceGroups = []aiv1.AIAgentResourceGroup{{Name: "a"}, {Name: ""}, {Name: "a"}}
	err := Normalize(agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.access.resourceGroups[1].name: Required value")
	assert.Contains(t, err.Error(), "Duplicate value")
}

// The label a Test Workflow joins a resource group with is refused on every AI
// kind rather than dropped, and the other labels are left alone.
func TestNormalize_AIKindsRefuseTheGroupLabel(t *testing.T) {
	grouped := metav1.ObjectMeta{Name: "x", Labels: map[string]string{ResourceGroupLabel: "payments", "kustomize.toolkit.fluxcd.io/name": "apps"}}
	for name, obj := range map[string]Object{
		"agent":    &aiv1.AIAgent{ObjectMeta: grouped, Spec: aiv1.AIAgentSpec{Instructions: "Do things."}},
		"server":   &aiv1.MCPServer{ObjectMeta: grouped, Spec: aiv1.MCPServerSpec{URL: "https://example.com/mcp"}},
		"provider": &aiv1.LLMProvider{ObjectMeta: grouped, Spec: aiv1.LLMProviderSpec{Provider: "openai"}},
	} {
		err := Normalize(obj)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "metadata.labels[testkube.io/group]", name)
		assert.NotContains(t, err.Error(), "fluxcd", name)
	}
	agent := &aiv1.AIAgent{ObjectMeta: metav1.ObjectMeta{Name: "x", Labels: map[string]string{"kustomize.toolkit.fluxcd.io/name": "apps"}}, Spec: aiv1.AIAgentSpec{Instructions: "Do things."}}
	require.NoError(t, Normalize(agent))
}

// An agent that leaves mcpServers out keeps its links, while mcpServers: []
// removes them, so nil and empty have to survive decoding, normalizing and the
// apply payload, as they do for skills.
func TestAIAgentMCPServers_AbsentAndEmptyDiffer(t *testing.T) {
	const head = "apiVersion: ai.testkube.io/v1alpha1\nkind: AIAgent\nmetadata:\n  name: helper\nspec:\n  instructions: Help.\n"
	for name, tc := range map[string]struct {
		yaml    string
		nil     bool
		payload string
	}{
		"absent": {yaml: head, nil: true},
		"null":   {yaml: head + "  mcpServers: null\n", nil: true},
		"empty":  {yaml: head + "  mcpServers: []\n", nil: false, payload: `"mcpServers":[]`},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := Decode(strings.NewReader(tc.yaml))
			require.NoError(t, err)
			require.Len(t, objects, 1)
			require.NoError(t, Normalize(objects[0]))
			agent := objects[0].(*aiv1.AIAgent)
			assert.Equal(t, tc.nil, agent.Spec.MCPServers == nil)

			payload, err := ApplyPayload(agent)
			require.NoError(t, err)
			if tc.payload == "" {
				assert.NotContains(t, string(payload), "mcpServers", "an absent list is not sent as null or []")
			} else {
				assert.Contains(t, string(payload), tc.payload)
			}
			sent, err := DecodeObject(payload)
			require.NoError(t, err)
			assert.Equal(t, tc.nil, sent.(*aiv1.AIAgent).Spec.MCPServers == nil, "the Control Plane reads what was applied")
		})
	}
}

// Credentials are taken as written, as the REST API and Test Workflows take
// them: in headers, query parameters and URLs, at the provider and the model.
func TestNormalize_AcceptsLiteralCredentials(t *testing.T) {
	p := &aiv1.LLMProvider{Spec: aiv1.LLMProviderSpec{
		Provider:     "azure",
		BaseURL:      "https://example.openai.azure.com/openai?sv=2024-01-01&sig=c2lnbmF0dXJl",
		ExtraHeaders: map[string]string{"X-Api-Key": "sk-live-123"},
		QueryParams:  map[string]string{"sig": "c2lnbmF0dXJl"},
		Models: []aiv1.LLMProviderModel{{
			Name:         "gpt-4o",
			URL:          "https://user:hunter2@llm.example.com/v1?api_key=sk-live-123",
			ExtraHeaders: map[string]string{"api-key": "sk-live-123"},
			QueryParams:  map[string]string{"key": "sk-live-123"},
		}},
	}}
	p.Name = "azure"
	require.NoError(t, Normalize(p))

	server := &aiv1.MCPServer{Spec: aiv1.MCPServerSpec{URL: "https://user:hunter2@example.com/mcp?token=hunter2"}}
	server.Name = "private"
	require.NoError(t, Normalize(server))
}

// An export redacts every value, and the placeholder applied back keeps the
// stored value, so it is accepted whatever the name.
func TestNormalize_AcceptsTheExportedPlaceholder(t *testing.T) {
	p := &aiv1.LLMProvider{Spec: aiv1.LLMProviderSpec{
		Provider:     "openai",
		ExtraHeaders: map[string]string{"X-Api-Key": RedactedValue, "anthropic-version": RedactedValue},
		Models:       []aiv1.LLMProviderModel{{Name: "gpt-4o", QueryParams: map[string]string{"key": RedactedValue, "sig": RedactedValue}}},
	}}
	p.Name = "openai"
	require.NoError(t, Normalize(p))

	p.Spec.ExtraHeaders = map[string]string{"Host": RedactedValue}
	require.Error(t, Normalize(p), "a header that is never forwarded stays refused")
}

// The Control Plane sends the key as Authorization and refuses the header, so
// the dry run refuses it too, at the provider and at a model.
func TestNormalize_LLMProviderRefusesAuthorizationHeader(t *testing.T) {
	for _, headers := range []map[string]string{{"Authorization": "Bearer sk-live-123"}, {"authorization": RedactedValue}} {
		p := &aiv1.LLMProvider{Spec: aiv1.LLMProviderSpec{Provider: "openai", ExtraHeaders: headers}}
		p.Name = "openai"
		err := Normalize(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is set from apiKey")
		assert.NotContains(t, err.Error(), "sk-live-123")

		p.Spec.ExtraHeaders = nil
		p.Spec.Models = []aiv1.LLMProviderModel{{Name: "gpt-4o", ExtraHeaders: headers}}
		require.Error(t, Normalize(p))
	}
}

func TestIsCredentialField(t *testing.T) {
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Api-Key", "api_key", "key", "x-goog-api-key", "X-Auth-Token", "client_secret", "password", "sig.key", "X-Amz-Signature", "sig", "SIG", "x-ms-sig", "key=sk-live-123", "X-Api-Key: sk-live-123", "sig=abc"} {
		assert.True(t, IsCredentialField(name), name)
	}
	for _, name := range []string{"anthropic-version", "api-version", "OpenAI-Organization", "x-request-id", "keyboard", "model", "signal", "sigma", "model=gpt-4o"} {
		assert.False(t, IsCredentialField(name), name)
	}
}

func TestCredentialNames(t *testing.T) {
	for value, want := range map[string][]string{
		"plain":                                             nil,
		`{{ credential("github-token") }}`:                  {"github-token"},
		`Bearer {{credential("github-token")}}`:             {"github-token"},
		`{{ credential("user") }}:{{ credential("pass") }}`: {"user", "pass"},
	} {
		names, ok := CredentialNames(value)
		assert.True(t, ok, value)
		assert.Equal(t, want, names, value)
	}
	for _, value := range []string{`{{ env("TOKEN") }}`, `{{ credential("bad name") }}`, `{{ credential("") }}`, `Bearer {{ credential("a") }} }}`, `{{ credential('a') }}`} {
		_, ok := CredentialNames(value)
		assert.False(t, ok, value)
	}
}

func TestAPIKeyCredential(t *testing.T) {
	for value, want := range map[string]string{
		`{{ credential("openai-key") }}`: "openai-key",
		` {{credential("openai-key")}} `: "openai-key",
	} {
		name, ok := APIKeyCredential(value)
		assert.True(t, ok, value)
		assert.Equal(t, want, name)
	}
	for _, value := range []string{"sk-live-123", "vault://kv/openai", "secret://openai-key", `Bearer {{ credential("openai-key") }}`, `{{ credential("a") }}{{ credential("b") }}`} {
		_, ok := APIKeyCredential(value)
		assert.False(t, ok, value)
	}
	assert.Equal(t, `{{ credential("openai-key") }}`, CredentialExpression("openai-key"))
}

func TestExpandCredentials(t *testing.T) {
	values := map[string]string{"user": "alice", "pass": "hunter2"}
	resolve := func(name string) (string, error) {
		if v, ok := values[name]; ok {
			return v, nil
		}
		return "", errors.New("no such credential")
	}
	got, err := ExpandCredentials(`Basic {{ credential("user") }}:{{credential("pass")}}`, resolve)
	require.NoError(t, err)
	assert.Equal(t, "Basic alice:hunter2", got)
	got, err = ExpandCredentials("2022-11-28", resolve)
	require.NoError(t, err)
	assert.Equal(t, "2022-11-28", got)
	_, err = ExpandCredentials(`Bearer {{ credential("missing") }}`, resolve)
	require.Error(t, err)
	_, err = ExpandCredentials(`{{ env("TOKEN") }}`, resolve)
	require.Error(t, err)
}

func TestValidateSignInHeaders(t *testing.T) {
	path := fieldPath("headers")
	assert.Empty(t, ValidateSignInHeaders(map[string]string{
		"Authorization":        `Bearer {{ credential("github-token") }}`,
		"X-Api-Key":            `{{ credential("api-key") }}`,
		"X-GitHub-Api-Version": "2022-11-28",
	}, path))
	for name, headers := range map[string]map[string]string{
		"literal key":      {"Authorization": "Bearer ghp_abc123"},
		"literal api key":  {"X-Api-Key": "ghp_abc123"},
		"other expression": {"X-Trace": `{{ env("ghp_abc123") }}`},
		"never sent":       {"Host": "example.com"},
		"empty name":       {"": "x"},
	} {
		errs := ValidateSignInHeaders(headers, path)
		require.NotEmpty(t, errs, name)
		assert.NotContains(t, errs.ToAggregate().Error(), "ghp_abc123", name)
	}
}

// A refused value may be the secret itself, and errors end up in terminal and
// CI logs, so none is echoed.
func TestRefusedSecretsAreNotEchoed(t *testing.T) {
	path := fieldPath("x")
	for _, key := range []string{"Bearer ghp_abc123", "vault://ghp_abc123", "ghp_abc123"} {
		p := &aiv1.LLMProvider{Spec: aiv1.LLMProviderSpec{Provider: "openai", APIKey: key}}
		p.Name = "openai"
		err := Normalize(p)
		require.Error(t, err, key)
		assert.NotContains(t, err.Error(), "ghp_abc123")
	}
	for _, raw := range []string{
		"ftp://user:hunter2@example.com/mcp",
		"//user:hunter2@example.com/mcp",
		"https://user:hunter2@exa mple.com/mcp",
	} {
		errs := validateHTTPURL(raw, path)
		require.NotEmpty(t, errs, raw)
		assert.NotContains(t, errs.ToAggregate().Error(), "hunter2", raw)
	}
	assert.Empty(t, validateHTTPURL("https://example.com/mcp?api-version=2024-10-21", path))
}

func TestProposeName(t *testing.T) {
	for display, want := range map[string]string{
		"GitHub MCP":              "github-mcp",
		"  Flaky Test Triager! ":  "flaky-test-triager",
		"Azure OpenAI (EU)":       "azure-openai-eu",
		"under_score.and.dots":    "under-score-and-dots",
		"!!!":                     "",
		strings.Repeat("ab-", 40): strings.TrimSuffix(strings.Repeat("ab-", 21), "-"),
	} {
		got := ProposeName(display)
		assert.Equal(t, want, got, display)
		if got != "" {
			assert.Empty(t, validateAIName(got, fieldPath("name")), got)
		}
	}
}

func TestSortForApply_DependenciesFirst(t *testing.T) {
	objects, err := decodeFile(t, "valid/ai-agent-full.yaml")
	require.NoError(t, err)
	board := &insightsv1.Board{}
	board.Name = "b"
	provider := &aiv1.LLMProvider{}
	provider.Name = "p"
	objects = append(objects, provider, board)

	SortForApply(objects)
	var order []string
	for _, obj := range objects {
		kind, err := KindOf(obj)
		require.NoError(t, err)
		order = append(order, kind.Kind+"/"+obj.GetName())
	}
	assert.Equal(t, []string{"Board/b", "LLMProvider/p", "MCPServer/github", "MCPServer/local-files", "AIAgent/flaky-test-triager"}, order)
}

func TestPrepareForApply_AIKinds(t *testing.T) {
	server := &aiv1.MCPServer{Status: aiv1.MCPServerStatus{Connection: "active", LastError: "boom"}}
	server.Name = "github"
	server.Status.ID = "abc"
	server.ResourceVersion = "7"
	prepared, err := PrepareForApply(server)
	require.NoError(t, err)
	got := prepared.(*aiv1.MCPServer)
	assert.Equal(t, aiv1.MCPServerStatus{}, got.Status)
	assert.Empty(t, got.ResourceVersion)
	assert.Equal(t, "active", server.Status.Connection, "the original is left alone")
	assert.Equal(t, "abc", StatusOf(server).ID)
}

func TestLookup_AIKinds(t *testing.T) {
	for _, name := range []string{"aiagent", "AIAgents", "mcpserver", "llmproviders"} {
		kind, ok := LookupName(name)
		require.True(t, ok, name)
		assert.Equal(t, ScopeEnvironment, kind.Scope)
	}
	resources := map[string]bool{}
	for _, k := range Kinds() {
		assert.False(t, resources[k.Resource], "resource %s is not unique", k.Resource)
		resources[k.Resource] = true
	}
}

func fieldPath(name string) *field.Path { return field.NewPath(name) }

// Only the Skills an agent always includes count against the budget: the
// others are read on demand.
func TestNormalize_AIAgentSkillBudgetCountsAlwaysIncluded(t *testing.T) {
	half := strings.Repeat("a", MaxAlwaysIncludedSkillsLength/2+1)
	agent := func(always bool) *aiv1.AIAgent {
		return &aiv1.AIAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "helper"},
			Spec: aiv1.AIAgentSpec{Instructions: "Help.", Skills: []aiv1.AIAgentSkill{
				{Name: "one", Description: "One.", Content: half, AlwaysInclude: true},
				{Name: "two", Description: "Two.", Content: half, AlwaysInclude: always},
			}},
		}
	}
	require.NoError(t, Normalize(agent(false)))
	err := Normalize(agent(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.skills")
	assert.Contains(t, err.Error(), "always included is over 32000 characters")
	assert.NotContains(t, err.Error(), half[:100], "the content is not echoed")
}

// An agent that leaves skills out keeps its Skills, while skills: [] removes
// them all, so the difference between nil and empty has to survive decoding,
// normalizing and the apply payload.
func TestAIAgentSkills_AbsentAndEmptyDiffer(t *testing.T) {
	const head = "apiVersion: ai.testkube.io/v1alpha1\nkind: AIAgent\nmetadata:\n  name: helper\nspec:\n  instructions: Help.\n"
	for name, tc := range map[string]struct {
		yaml    string
		nil     bool
		payload string
	}{
		"absent": {yaml: head, nil: true, payload: ""},
		"null":   {yaml: head + "  skills: null\n", nil: true, payload: ""},
		"empty":  {yaml: head + "  skills: []\n", nil: false, payload: `"skills":[]`},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := Decode(strings.NewReader(tc.yaml))
			require.NoError(t, err)
			require.Len(t, objects, 1)
			require.NoError(t, Normalize(objects[0]))
			agent := objects[0].(*aiv1.AIAgent)
			assert.Equal(t, tc.nil, agent.Spec.Skills == nil)
			assert.Empty(t, agent.Spec.Skills)

			payload, err := ApplyPayload(agent)
			require.NoError(t, err)
			if tc.payload == "" {
				assert.NotContains(t, string(payload), "skills", "an absent list is not sent as null or []")
			} else {
				assert.Contains(t, string(payload), tc.payload)
			}
			sent, err := DecodeObject(payload)
			require.NoError(t, err)
			assert.Equal(t, tc.nil, sent.(*aiv1.AIAgent).Spec.Skills == nil, "the Control Plane reads what was applied")
		})
	}
}

// The lengths are counted in characters, and the errors say so without
// echoing the value.
func TestNormalize_AIAgentSkillLengthsInCharacters(t *testing.T) {
	agent := func(description, content string) *aiv1.AIAgent {
		return &aiv1.AIAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "helper"},
			Spec: aiv1.AIAgentSpec{Instructions: "Help.", Skills: []aiv1.AIAgentSkill{
				{Name: "one", Description: description, Content: content},
			}},
		}
	}
	// Two bytes a character: over the limit in bytes, within it in characters.
	require.NoError(t, Normalize(agent(strings.Repeat("é", MaxSkillDescriptionLength), strings.Repeat("é", MaxSkillContentLength))))

	content := strings.Repeat("b", MaxSkillContentLength+1)
	description := strings.Repeat("d", MaxSkillDescriptionLength+1)
	err := Normalize(agent(description, content))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.skills[0].content: Invalid value: may not be more than 32000 characters")
	assert.Contains(t, err.Error(), "spec.skills[0].description: Invalid value: may not be more than 1024 characters")
	assert.NotContains(t, err.Error(), "bytes")
	assert.NotContains(t, err.Error(), content[:100], "the content is not echoed")
	assert.NotContains(t, err.Error(), description[:100], "the description is not echoed")
}
