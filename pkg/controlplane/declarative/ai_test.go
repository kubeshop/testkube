package declarative

import (
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
		Access:       &aiv1.AIAgentAccess{Roles: []string{"write", "read"}, ResourceGroups: []string{"b", "a"}},
	}}
	agent.Name = "triage"
	require.NoError(t, Normalize(agent))
	assert.Equal(t, "Triage", agent.Spec.DisplayName)
	assert.Equal(t, []string{"read", "write"}, agent.Spec.Access.Roles, "sorted as the Control Plane returns them")
	assert.Equal(t, []string{"a", "b"}, agent.Spec.Access.ResourceGroups)
}

// Credentials are taken as written, as the REST API and Test Workflows take
// them: in headers, query parameters and URLs, at the provider and the model.
func TestNormalize_AcceptsLiteralCredentials(t *testing.T) {
	p := &aiv1.LLMProvider{Spec: aiv1.LLMProviderSpec{
		Provider:     "azure",
		BaseURL:      "https://example.openai.azure.com/openai?sv=2024-01-01&sig=c2lnbmF0dXJl",
		ExtraHeaders: map[string]string{"X-Api-Key": "sk-live-123", "Authorization": "Bearer sk-live-123"},
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

func TestCredentialReference(t *testing.T) {
	for ref, want := range map[string][2]string{
		"secret://github-token": {"github-token", "credential"},
		"github-token":          {"github-token", "credential"},
		"vault://kv/openai":     {"kv/openai", "vault"},
	} {
		name, source := ParseCredentialReference(ref)
		assert.Equal(t, want, [2]string{name, source}, ref)
	}
	path := fieldPath("credentialReference")
	assert.Empty(t, validateCredentialReference("secret://github-token", false, path))
	assert.Empty(t, validateCredentialReference("vault://kv/openai", true, path))
	assert.NotEmpty(t, validateCredentialReference("vault://kv/openai", false, path), "only providers read Vault")
	assert.NotEmpty(t, validateCredentialReference("secret://", false, path))
	assert.NotEmpty(t, validateCredentialReference("sk-live 123", false, path))
}

// A refused value may be the secret itself, and errors end up in terminal and
// CI logs, so none is echoed.
func TestRefusedSecretsAreNotEchoed(t *testing.T) {
	path := fieldPath("x")
	for _, ref := range []string{"Bearer ghp_abc123", "vault://ghp_abc123"} {
		errs := validateCredentialReference(ref, false, path)
		require.NotEmpty(t, errs, ref)
		assert.NotContains(t, errs.ToAggregate().Error(), "ghp_abc123")
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
