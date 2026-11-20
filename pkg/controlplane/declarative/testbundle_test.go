package declarative

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
)

func testBundle(spec testbundlesv1.TestBundleSpec) *testbundlesv1.TestBundle {
	return &testbundlesv1.TestBundle{ObjectMeta: metav1.ObjectMeta{Name: "checkout"}, Spec: spec}
}

func TestNormalize_TestBundleDefaultsAndSorts(t *testing.T) {
	b := testBundle(testbundlesv1.TestBundleSpec{AIAgents: &[]string{"triager", "helper"}})
	require.NoError(t, Normalize(b))
	assert.Equal(t, "checkout", b.Spec.DisplayName)
	assert.Equal(t, []string{"helper", "triager"}, *b.Spec.AIAgents, "sorted as an export lists them")
}

// An omitted list keeps the bundle's AI Agents, and an empty one clears them,
// so the difference has to reach the Control Plane.
func TestApplyPayload_TestBundleKeepsOmittedAndEmptyAIAgentsApart(t *testing.T) {
	for name, tc := range map[string]struct {
		agents *[]string
		want   string
	}{
		"omitted": {nil, ""},
		"empty":   {&[]string{}, `"aiAgents":[]`},
	} {
		t.Run(name, func(t *testing.T) {
			payload, err := ApplyPayload(testBundle(testbundlesv1.TestBundleSpec{AIAgents: tc.agents}))
			require.NoError(t, err)
			if tc.want == "" {
				assert.NotContains(t, string(payload), "aiAgents")
				return
			}
			assert.Contains(t, string(payload), tc.want)

			var back testbundlesv1.TestBundle
			require.NoError(t, json.Unmarshal(payload, &back))
			require.NotNil(t, back.Spec.AIAgents)
			assert.Empty(t, *back.Spec.AIAgents)
		})
	}
}

// Only the Skills always included count toward the budget, as an AI Agent's.
func TestNormalize_TestBundleSkillBudgetCountsAlwaysIncluded(t *testing.T) {
	half := strings.Repeat("a", MaxAlwaysIncludedSkillsLength/2+1)
	for _, always := range []bool{false, true} {
		b := testBundle(testbundlesv1.TestBundleSpec{Skills: []testbundlesv1.TestBundleSkill{
			{Name: "one", Description: "One.", Content: half, AlwaysInclude: true},
			{Name: "two", Description: "Two.", Content: half, AlwaysInclude: always},
		}})
		err := Normalize(b)
		if !always {
			assert.NoError(t, err, "a Skill read on demand is not in every prompt")
			continue
		}
		require.Error(t, err)
		assert.Contains(t, err.Error(), "spec.skills")
		assert.Contains(t, err.Error(), "over 32000 characters")
		assert.NotContains(t, err.Error(), half[:100], "the content is not echoed")
	}
}

// alwaysInclude reaches the Control Plane when set and is left out when not,
// so an exported on-demand Skill applies back on demand.
func TestApplyPayload_TestBundleSkillAlwaysInclude(t *testing.T) {
	payload, err := ApplyPayload(testBundle(testbundlesv1.TestBundleSpec{Skills: []testbundlesv1.TestBundleSkill{
		{Name: "always", Description: "Always.", Content: "a", AlwaysInclude: true},
		{Name: "on-demand", Description: "On demand.", Content: "b"},
	}}))
	require.NoError(t, err)
	var back testbundlesv1.TestBundle
	require.NoError(t, json.Unmarshal(payload, &back))
	require.Len(t, back.Spec.Skills, 2)
	assert.True(t, back.Spec.Skills[0].AlwaysInclude)
	assert.False(t, back.Spec.Skills[1].AlwaysInclude)
	assert.Equal(t, 1, strings.Count(string(payload), "alwaysInclude"))
}

func TestNormalize_TestBundleGitURI(t *testing.T) {
	for uri, ok := range map[string]bool{
		"https://github.com/acme/checkout.git":          true,
		"git@github.com:acme/checkout.git":              true,
		"ssh://git@github.com/acme/checkout.git":        true,
		"https://github.com/acme/checkout.git?ref=main": true,
		"github.com:acme/checkout.git":                  true,
		"[::1]:acme/checkout.git":                       true,
		"git@git.example.com:team/repo@v2.git":          true,
		"git@git.example.com:repo@v2.git":               true,
		"git.example.com:team/repo@v2.git":              true,
		// A user over ssh is not a secret; a password anywhere, and a user
		// over http or https (usually a token), are.
		"ssh://git@github.com:2222/acme/checkout.git":                    true,
		"https://ghp_abc123@github.com/acme/checkout.git":                false,
		"https://x-access-token:ghp_abc123@github.com/acme/checkout.git": false,
		"http://ghp_abc123@git.example.com/acme/checkout.git":            false,
		"HTTPS://ghp_abc123@github.com/acme/checkout.git":                false,
		"ssh://git:hunter2@github.com/acme/checkout.git":                 false,
		"https://:ghp_abc123@github.com/acme/checkout.git":               false,
		// url.Parse cannot read these; the refusal does not echo them.
		"https://x-access-token:ghp_abc123@github.com/acme/repo%zz.git": false,
		"https://ghp_abc123@github.com/acme/repo%zz.git":                false,
		// Neither a URL with a host nor scp-like.
		"//git:hunter2@github.com/acme/checkout.git": false,
		"https:ghp_abc123@github.com/acme/checkout":  false,
		"acme/checkout": false,
	} {
		t.Run(uri, func(t *testing.T) {
			err := Normalize(testBundle(testbundlesv1.TestBundleSpec{GitSource: &testbundlesv1.TestBundleGitSource{URI: uri}}))
			if ok {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "ghp_abc123")
			assert.NotContains(t, err.Error(), "hunter2")
		})
	}
	err := Normalize(testBundle(testbundlesv1.TestBundleSpec{GitSource: &testbundlesv1.TestBundleGitSource{}}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.gitSource.uri: Required")
}

func TestGitURIWithoutCredentials(t *testing.T) {
	for raw, want := range map[string]string{
		"https://ghp_abc123@github.com/acme/checkout.git":                "https://github.com/acme/checkout.git",
		"https://x-access-token:ghp_abc123@github.com/acme/checkout.git": "https://github.com/acme/checkout.git",
		"ssh://git:hunter2@github.com/acme/checkout.git":                 "ssh://git@github.com/acme/checkout.git",
		"ssh://git@github.com/acme/checkout.git":                         "ssh://git@github.com/acme/checkout.git",
		"git@github.com:acme/checkout.git":                               "git@github.com:acme/checkout.git",
		"https://github.com/acme/checkout.git":                           "https://github.com/acme/checkout.git",
		"https://github.com/acme/checkout.git?ref=a@b":                   "https://github.com/acme/checkout.git?ref=a@b",
		"https://:ghp_abc123@github.com:8443/acme/checkout.git":          "https://github.com:8443/acme/checkout.git",
	} {
		t.Run(raw, func(t *testing.T) {
			got := GitURIWithoutCredentials(raw)
			assert.Equal(t, want, got)
			require.NoError(t, Normalize(testBundle(testbundlesv1.TestBundleSpec{GitSource: &testbundlesv1.TestBundleGitSource{URI: got}})), "the export applies back")
		})
	}
	// A URL url.Parse cannot read still loses its token, though the manifest
	// refuses what is left.
	assert.Equal(t, "https://github.com/acme/repo%zz.git", GitURIWithoutCredentials("https://x-access-token:ghp_abc123@github.com/acme/repo%zz.git"))
}

func TestSortForApply_TestBundleAfterItsAIAgents(t *testing.T) {
	objects, err := Decode(strings.NewReader(`apiVersion: testbundles.testkube.io/v1alpha1
kind: TestBundle
metadata:
  name: checkout
spec:
  aiAgents: [triager]
---
apiVersion: ai.testkube.io/v1alpha1
kind: AIAgent
metadata:
  name: triager
spec:
  instructions: Triage failures.
`))
	require.NoError(t, err)
	SortForApply(objects)
	assert.Equal(t, []string{"triager", "checkout"}, []string{objects[0].GetName(), objects[1].GetName()})
}

// A bundle that leaves contexts out keeps its Attached Context, while
// contexts: [] removes it all, as with skills.
func TestTestBundleContexts_AbsentAndEmptyDiffer(t *testing.T) {
	const head = "apiVersion: testbundles.testkube.io/v1alpha1\nkind: TestBundle\nmetadata:\n  name: checkout\nspec:\n  displayName: Checkout\n"
	for name, tc := range map[string]struct {
		yaml    string
		nil     bool
		payload string
	}{
		"absent": {yaml: head, nil: true},
		"null":   {yaml: head + "  contexts: null\n", nil: true},
		"empty":  {yaml: head + "  contexts: []\n", nil: false, payload: `"contexts":[]`},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := Decode(strings.NewReader(tc.yaml))
			require.NoError(t, err)
			require.Len(t, objects, 1)
			require.NoError(t, Normalize(objects[0]))
			bundle := objects[0].(*testbundlesv1.TestBundle)
			assert.Equal(t, tc.nil, bundle.Spec.Contexts == nil)

			payload, err := ApplyPayload(bundle)
			require.NoError(t, err)
			if tc.payload == "" {
				assert.NotContains(t, string(payload), "contexts", "an absent list is not sent as null or []")
			} else {
				assert.Contains(t, string(payload), tc.payload)
			}
			sent, err := DecodeObject(payload)
			require.NoError(t, err)
			assert.Equal(t, tc.nil, sent.(*testbundlesv1.TestBundle).Spec.Contexts == nil, "the Control Plane reads what was applied")
		})
	}
}

// Attached Context titles are trimmed and may repeat, as in the dashboard.
func TestNormalize_TestBundleContextTitles(t *testing.T) {
	b := testBundle(testbundlesv1.TestBundleSpec{Contexts: []testbundlesv1.TestBundleContext{
		{Title: "  Notes ", Content: "first"},
		{Title: "Notes", Content: "second"},
	}})
	require.NoError(t, Normalize(b))
	assert.Equal(t, []testbundlesv1.TestBundleContext{{Title: "Notes", Content: "first"}, {Title: "Notes", Content: "second"}}, b.Spec.Contexts)
}

// A bundle that leaves skills out keeps its Skills, while skills: [] removes
// them all, so the difference has to survive decoding, normalizing and the
// apply payload.
func TestTestBundleSkills_AbsentAndEmptyDiffer(t *testing.T) {
	const head = "apiVersion: testbundles.testkube.io/v1alpha1\nkind: TestBundle\nmetadata:\n  name: checkout\nspec:\n  displayName: Checkout\n"
	for name, tc := range map[string]struct {
		yaml    string
		nil     bool
		payload string
	}{
		"absent": {yaml: head, nil: true},
		"null":   {yaml: head + "  skills: null\n", nil: true},
		"empty":  {yaml: head + "  skills: []\n", nil: false, payload: `"skills":[]`},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := Decode(strings.NewReader(tc.yaml))
			require.NoError(t, err)
			require.Len(t, objects, 1)
			require.NoError(t, Normalize(objects[0]))
			bundle := objects[0].(*testbundlesv1.TestBundle)
			assert.Equal(t, tc.nil, bundle.Spec.Skills == nil)
			assert.Empty(t, bundle.Spec.Skills)

			payload, err := ApplyPayload(bundle)
			require.NoError(t, err)
			if tc.payload == "" {
				assert.NotContains(t, string(payload), "skills", "an absent list is not sent as null or []")
			} else {
				assert.Contains(t, string(payload), tc.payload)
			}
			sent, err := DecodeObject(payload)
			require.NoError(t, err)
			assert.Equal(t, tc.nil, sent.(*testbundlesv1.TestBundle).Spec.Skills == nil, "the Control Plane reads what was applied")
		})
	}
}
