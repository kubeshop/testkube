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

func TestNormalize_TestBundleSkillsShareOneBudget(t *testing.T) {
	half := strings.Repeat("a", MaxAlwaysIncludedSkillsLength/2+1)
	b := testBundle(testbundlesv1.TestBundleSpec{Skills: []testbundlesv1.TestBundleSkill{
		{Name: "one", Description: "One.", Content: half},
		{Name: "two", Description: "Two.", Content: half},
	}})
	err := Normalize(b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.skills")
	assert.Contains(t, err.Error(), "over 32000 characters")
	assert.NotContains(t, err.Error(), half[:100], "the content is not echoed")
}

func TestNormalize_TestBundleGitURI(t *testing.T) {
	for uri, ok := range map[string]bool{
		"https://github.com/acme/checkout.git":                 true,
		"git@github.com:acme/checkout.git":                     true,
		"ssh://git@github.com/acme/checkout.git":               true,
		"https://ghp_abc123@github.com/acme/checkout.git":      false,
		"ssh://git:hunter2@github.com/acme/checkout.git":       false,
		"https://github.com/acme/checkout.git?access_token=x1": false,
		"https://github.com/acme/checkout.git?ref=main":        true,
		// URL.Query would drop every parameter of this query, the token too.
		"https://github.com/acme/checkout.git?access_token=ghp_abc123;other=value": false,
		"github.com:acme/checkout.git":                                             true,
		"[::1]:acme/checkout.git":                                                  true,
		// url.Parse cannot read these, so their credentials could not be checked.
		"https://x-access-token:ghp_abc123@github.com/acme/repo%zz.git": false,
		"https://ghp_abc123@github.com/acme/repo%zz.git":                false,
		"git:hunter2@github.com:acme/checkout.git":                      false,
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
