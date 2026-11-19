package declarative

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
)

func newSkill(name string, spec aiv1.SkillSpec) *aiv1.Skill {
	return &aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: spec}
}

// A Skill is normalized so an export, written as the Control Plane stores it,
// applies without a diff: text is trimmed, allowed-tools is space separated
// however the YAML wrapped it, and a testkube block with both settings off is
// dropped, as leaving it out means the same.
func TestNormalize_Skill(t *testing.T) {
	s := newSkill("k6-load-testing", aiv1.SkillSpec{
		Description:   "  Use for k6.  ",
		License:       " MIT ",
		Compatibility: " Needs k6. ",
		AllowedTools:  "Bash(k6:*)\n  Read ",
		Content:       "# k6\n",
		Testkube:      &aiv1.SkillTestkubeSettings{},
	})
	require.NoError(t, Normalize(s))
	assert.Equal(t, aiv1.SkillSpec{
		Description:   "Use for k6.",
		License:       "MIT",
		Compatibility: "Needs k6.",
		AllowedTools:  "Bash(k6:*) Read",
		Content:       "# k6\n",
	}, s.Spec)

	s.Spec.Testkube = &aiv1.SkillTestkubeSettings{AddToNewTestBundles: true}
	require.NoError(t, Normalize(s))
	assert.Equal(t, &aiv1.SkillTestkubeSettings{AddToNewTestBundles: true}, s.Spec.Testkube, "a setting that is on keeps the block")
}

// metadata.name names a custom resource and a SKILL.md at once, so it follows
// both rules: a DNS-1123 label of at most 63 characters and an Agent Skills
// name with no consecutive hyphens.
func TestNormalize_SkillName(t *testing.T) {
	for name, valid := range map[string]bool{
		"k6-load-testing":       true,
		"a":                     true,
		strings.Repeat("a", 63): true,
		strings.Repeat("a", 64): false,
		"k6--load":              false,
		"-k6":                   false,
		"K6":                    false,
		"k6_load":               false,
		"":                      false,
	} {
		err := Normalize(newSkill(name, aiv1.SkillSpec{Description: "D.", Content: "C."}))
		if valid {
			assert.NoError(t, err, name)
			continue
		}
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "metadata.name", name)
	}
}

// The limits are the Agent Skills spec's, counted in characters, and the
// errors do not echo the value.
func TestNormalize_SkillLimits(t *testing.T) {
	require.NoError(t, Normalize(newSkill("s", aiv1.SkillSpec{
		Description:   strings.Repeat("é", MaxSkillDescriptionLength),
		Compatibility: strings.Repeat("é", MaxSkillCompatibilityLength),
		Content:       strings.Repeat("é", MaxSkillContentLength),
	})))

	description := strings.Repeat("d", MaxSkillDescriptionLength+1)
	compatibility := strings.Repeat("c", MaxSkillCompatibilityLength+1)
	content := strings.Repeat("b", MaxSkillContentLength+1)
	err := Normalize(newSkill("s", aiv1.SkillSpec{Description: description, Compatibility: compatibility, Content: content}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.description: Invalid value: may not be more than 1024 characters")
	assert.Contains(t, err.Error(), "spec.compatibility: Invalid value: may not be more than 500 characters")
	assert.Contains(t, err.Error(), "spec.content: Invalid value: may not be more than 32000 characters")
	for _, value := range []string{description, compatibility, content} {
		assert.NotContains(t, err.Error(), value[:100])
	}
}

// Only a GitOps runner can read a ConfigMap, so a skillFrom that reaches
// Normalize, as in testkube apply or a PUT to the Control Plane, is refused,
// and so is any field the SKILL.md would give.
func TestNormalize_SkillFromIsRefusedUnresolved(t *testing.T) {
	from := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "SKILL.md"}}
	err := Normalize(newSkill("k6", aiv1.SkillSpec{SkillFrom: from, Testkube: &aiv1.SkillTestkubeSettings{AlwaysInclude: true}}))
	require.Error(t, err)
	assert.Equal(t, `spec.skillFrom: Forbidden: only a GitOps runner reads a SKILL.md from a ConfigMap in its cluster; testkube apply takes the Skill inline`, err.Error(),
		"testkube settings stay in the manifest")

	err = Normalize(newSkill("k6", aiv1.SkillSpec{
		SkillFrom: from, Description: "D.", License: "MIT", Compatibility: "C.",
		Metadata: map[string]string{"a": "b"}, AllowedTools: "Read", Content: "C.",
	}))
	require.Error(t, err)
	for _, key := range []string{"description", "license", "compatibility", "metadata", "allowed-tools", "content"} {
		assert.Contains(t, err.Error(), "spec."+key+": Forbidden: comes from the SKILL.md that skillFrom reads", key)
	}

	for name, tc := range map[string]struct {
		from *aiv1.SkillSource
		want string
	}{
		"no ref":      {&aiv1.SkillSource{}, "spec.skillFrom.configMapKeyRef: Required value"},
		"no name":     {&aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Key: "SKILL.md"}}, "spec.skillFrom.configMapKeyRef.name: Required value"},
		"bad name":    {&aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "My Skills", Key: "SKILL.md"}}, `spec.skillFrom.configMapKeyRef.name: Invalid value: "My Skills"`},
		"bad key":     {&aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "a/SKILL.md"}}, `spec.skillFrom.configMapKeyRef.key: Invalid value: "a/SKILL.md"`},
		"missing key": {&aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills"}}, "spec.skillFrom.configMapKeyRef.key: Required value"},
	} {
		err := Normalize(newSkill("k6", aiv1.SkillSpec{SkillFrom: tc.from}))
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), tc.want, name)
	}
}

func agentWithRefs(skills []aiv1.AIAgentSkill, refs []aiv1.AIAgentSkillRef) *aiv1.AIAgent {
	return &aiv1.AIAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "helper"},
		Spec:       aiv1.AIAgentSpec{Instructions: "Help.", Skills: skills, SkillRefs: refs},
	}
}

// skillRefs name library Skills: each name once whatever its scope, since the
// agent knows its Skills by name, none the name of one of its own Skills
// (which the Control Plane answers with a 422), and in a known scope. The
// order is the agent's, so it is kept.
func TestNormalize_AIAgentSkillRefs(t *testing.T) {
	own := []aiv1.AIAgentSkill{{Name: "notes", Description: "Notes.", Content: "Some notes."}}
	refs := []aiv1.AIAgentSkillRef{
		{Name: " release-notes "},
		{Name: "k6-load-testing", Scope: aiv1.SkillScopeEnvironment},
		{Name: "company-style", Scope: aiv1.SkillScopeOrganization},
	}
	agent := agentWithRefs(own, refs)
	require.NoError(t, Normalize(agent))
	assert.Equal(t, []aiv1.AIAgentSkillRef{
		{Name: "release-notes"},
		{Name: "k6-load-testing", Scope: aiv1.SkillScopeEnvironment},
		{Name: "company-style", Scope: aiv1.SkillScopeOrganization},
	}, agent.Spec.SkillRefs, "trimmed, in manifest order, the default scope left empty")

	for name, tc := range map[string]struct {
		own  []aiv1.AIAgentSkill
		refs []aiv1.AIAgentSkillRef
		want string
	}{
		"clash with an own Skill": {own, []aiv1.AIAgentSkillRef{{Name: "notes"}}, `spec.skillRefs[0].name: Invalid value: "notes": is also the name of one of the agent's own Skills`},
		"clash after trimming":    {[]aiv1.AIAgentSkill{{Name: " notes ", Description: "N.", Content: "N."}}, []aiv1.AIAgentSkillRef{{Name: "notes "}}, `spec.skillRefs[0].name: Invalid value: "notes"`},
		"same name, two scopes":   {nil, []aiv1.AIAgentSkillRef{{Name: "style"}, {Name: "style", Scope: aiv1.SkillScopeOrganization}}, `spec.skillRefs[1].name: Duplicate value: "style"`},
		"no name":                 {nil, []aiv1.AIAgentSkillRef{{Scope: aiv1.SkillScopeOrganization}}, "spec.skillRefs[0].name: Required value"},
		"not a Skill name":        {nil, []aiv1.AIAgentSkillRef{{Name: "Style"}}, `spec.skillRefs[0].name: Invalid value: "Style"`},
		"unknown scope":           {nil, []aiv1.AIAgentSkillRef{{Name: "style", Scope: "cluster"}}, `spec.skillRefs[0].scope: Unsupported value: "cluster"`},
	} {
		err := Normalize(agentWithRefs(tc.own, tc.refs))
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), tc.want, name)
	}
}

// skillRefs left out keeps the agent's links, while skillRefs: [] unlinks
// them all, so nil and empty must reach the Control Plane apart.
func TestAIAgentSkillRefs_AbsentAndEmptyDiffer(t *testing.T) {
	const head = "apiVersion: ai.testkube.io/v1alpha1\nkind: AIAgent\nmetadata:\n  name: helper\nspec:\n  instructions: Help.\n"
	for name, tc := range map[string]struct {
		yaml    string
		nil     bool
		payload string
	}{
		"absent": {yaml: head, nil: true},
		"null":   {yaml: head + "  skillRefs: null\n", nil: true},
		"empty":  {yaml: head + "  skillRefs: []\n", payload: `"skillRefs":[]`},
		"one":    {yaml: head + "  skillRefs:\n    - name: k6\n", payload: `"skillRefs":[{"name":"k6"}]`},
	} {
		t.Run(name, func(t *testing.T) {
			objects, err := Decode(strings.NewReader(tc.yaml))
			require.NoError(t, err)
			require.NoError(t, Normalize(objects[0]))
			agent := objects[0].(*aiv1.AIAgent)
			assert.Equal(t, tc.nil, agent.Spec.SkillRefs == nil)

			payload, err := ApplyPayload(agent)
			require.NoError(t, err)
			if tc.payload == "" {
				assert.NotContains(t, string(payload), "skillRefs", "an absent list is not sent as null or []")
			} else {
				assert.Contains(t, string(payload), tc.payload)
			}
			sent, err := DecodeObject(payload)
			require.NoError(t, err)
			assert.Equal(t, tc.nil, sent.(*aiv1.AIAgent).Spec.SkillRefs == nil, "the Control Plane reads what was applied")
		})
	}
}

// An agent's own Skill with skillFrom is refused unresolved too, and so are
// the fields its SKILL.md gives; alwaysInclude stays in the manifest.
func TestNormalize_AIAgentOwnSkillFromIsRefusedUnresolved(t *testing.T) {
	from := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "SKILL.md"}}
	err := Normalize(agentWithRefs([]aiv1.AIAgentSkill{{Name: "triage", SkillFrom: from, AlwaysInclude: true}}, nil))
	require.Error(t, err)
	assert.Equal(t, "spec.skills[0].skillFrom: Forbidden: only a GitOps runner reads a SKILL.md from a ConfigMap in its cluster; testkube apply takes the Skill inline", err.Error())

	err = Normalize(agentWithRefs([]aiv1.AIAgentSkill{{Name: "triage", SkillFrom: from, Description: "D.", Content: "C.", Metadata: map[string]string{"a": "b"}}}, nil))
	require.Error(t, err)
	for _, key := range []string{"description", "content", "metadata"} {
		assert.Contains(t, err.Error(), "spec.skills[0]."+key+": Forbidden: comes from the SKILL.md", key)
	}
}

const k6SkillMarkdown = `---
name: k6-load-testing
description: >-
  Use when writing or reviewing
  a k6 load test.
license: Apache-2.0
compatibility: Needs k6 0.50 or later.
metadata:
  author: qa-team
  version: "1.2"
allowed-tools: Bash(k6:*) Read
argument-hint: "[script]"
---

# k6 load testing

Start from a smoke test.
`

func TestParseSkillMarkdown(t *testing.T) {
	want := SkillMarkdown{
		Name:          "k6-load-testing",
		Description:   "Use when writing or reviewing a k6 load test.",
		License:       "Apache-2.0",
		Compatibility: "Needs k6 0.50 or later.",
		Metadata:      map[string]string{"author": "qa-team", "version": "1.2"},
		AllowedTools:  "Bash(k6:*) Read",
		Body:          "# k6 load testing\n\nStart from a smoke test.\n",
	}
	md, err := ParseSkillMarkdown(k6SkillMarkdown)
	require.NoError(t, err)
	assert.Equal(t, want, md, "a key outside the spec, such as argument-hint, is ignored")

	md, err = ParseSkillMarkdown("\uFEFF" + strings.ReplaceAll(k6SkillMarkdown, "\n", "\r\n"))
	require.NoError(t, err, "a byte order mark and Windows line endings are read")
	assert.Equal(t, want, md)

	md, err = ParseSkillMarkdown("---\nname: a\ndescription: A.\n---\nBody")
	require.NoError(t, err)
	assert.Equal(t, "Body", md.Body)
	assert.Nil(t, md.Metadata)

	for name, tc := range map[string]struct{ data, want string }{
		"no frontmatter":       {"# k6\n", "starts with YAML frontmatter"},
		"unclosed frontmatter": {"---\nname: a\n# k6\n", "no closing --- line"},
		"not YAML":             {"---\nname: [a\n---\n", "frontmatter is not valid"},
		"number metadata":      {"---\nname: a\nmetadata:\n  version: 1.0\n---\n", `metadata "version" is not a string; quote it`},
		"bad allowed-tools":    {"---\nname: a\nallowed-tools:\n  read: true\n---\n", "allowed-tools is not a space-separated string"},
		"allowed-tools list":   {"---\nname: a\nallowed-tools:\n  - Read\n  - Bash(git:*)\n---\n", "allowed-tools is not a space-separated string"},
	} {
		_, err := ParseSkillMarkdown(tc.data)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), tc.want, name)
	}
}

// configMaps is a SkillSourceReader over ConfigMaps by name, and records what
// it was asked for.
type configMaps struct {
	data  map[string]map[string]string
	asked []aiv1.ConfigMapKeyReference
}

var errNoConfigMap = errors.New("configmaps \"x\" not found")

func (c *configMaps) read(ref aiv1.ConfigMapKeyReference) (string, error) {
	c.asked = append(c.asked, ref)
	value, ok := c.data[ref.Name][ref.Key]
	if !ok {
		return "", errNoConfigMap
	}
	return value, nil
}

// A Skill read from a SKILL.md is sent as if written inline: every key of the
// frontmatter, the body as content, and its own testkube settings kept.
func TestResolveSkillSources_Skill(t *testing.T) {
	cms := &configMaps{data: map[string]map[string]string{"skills": {"SKILL.md": k6SkillMarkdown}}}
	s := newSkill("k6-load-testing", aiv1.SkillSpec{
		SkillFrom: &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "SKILL.md"}},
		Testkube:  &aiv1.SkillTestkubeSettings{AlwaysInclude: true},
	})
	require.NoError(t, ResolveSkillSources(s, cms.read))
	assert.Equal(t, aiv1.SkillSpec{
		Description:   "Use when writing or reviewing a k6 load test.",
		License:       "Apache-2.0",
		Compatibility: "Needs k6 0.50 or later.",
		Metadata:      map[string]string{"author": "qa-team", "version": "1.2"},
		AllowedTools:  "Bash(k6:*) Read",
		Content:       "# k6 load testing\n\nStart from a smoke test.\n",
		Testkube:      &aiv1.SkillTestkubeSettings{AlwaysInclude: true},
	}, s.Spec)
	require.NoError(t, Normalize(s), "the resolved Skill is valid as written inline")

	inline := newSkill("inline", aiv1.SkillSpec{Description: "D.", Content: "C."})
	require.NoError(t, ResolveSkillSources(inline, cms.read))
	assert.Len(t, cms.asked, 1, "a Skill without skillFrom reads nothing")
}

// A SKILL.md that names another Skill, or does not fit, is fixed only by a
// change, so it is ErrInvalidSkillSource; a failed read, such as a ConfigMap
// Git has not brought yet, is not, so the runner retries it.
func TestResolveSkillSources_Errors(t *testing.T) {
	ref := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "SKILL.md"}}
	cms := &configMaps{data: map[string]map[string]string{"skills": {
		"SKILL.md": k6SkillMarkdown,
	}}}

	err := ResolveSkillSources(newSkill("other-name", aiv1.SkillSpec{SkillFrom: ref}), cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	assert.Contains(t, err.Error(), `spec.skillFrom: Invalid value: "k6-load-testing": the SKILL.md must be named as the Skill is: other-name`)

	cms.data["skills"]["SKILL.md"] = "---\ndescription: D.\n---\nBody"
	err = ResolveSkillSources(newSkill("k6-load-testing", aiv1.SkillSpec{SkillFrom: ref}), cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	assert.Contains(t, err.Error(), "the SKILL.md needs a name: k6-load-testing")

	cms.data["skills"]["SKILL.md"] = "# no frontmatter"
	err = ResolveSkillSources(newSkill("k6-load-testing", aiv1.SkillSpec{SkillFrom: ref}), cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	assert.Contains(t, err.Error(), `spec.skillFrom: key "SKILL.md" of ConfigMap "skills": a SKILL.md starts with YAML frontmatter`)

	missing := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "not-yet", Key: "SKILL.md"}}
	err = ResolveSkillSources(newSkill("k6-load-testing", aiv1.SkillSpec{SkillFrom: missing}), cms.read)
	require.ErrorIs(t, err, errNoConfigMap)
	assert.NotErrorIs(t, err, ErrInvalidSkillSource)

	asked := len(cms.asked)
	err = ResolveSkillSources(newSkill("k6-load-testing", aiv1.SkillSpec{SkillFrom: &aiv1.SkillSource{}}), cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	assert.Len(t, cms.asked, asked, "a malformed skillFrom reads nothing")
}

// An agent's own Skill takes the description, metadata and body of its
// SKILL.md, which must carry its name; its alwaysInclude stays.
func TestResolveSkillSources_AIAgentOwnSkills(t *testing.T) {
	cms := &configMaps{data: map[string]map[string]string{"skills": {"k6.md": k6SkillMarkdown}}}
	from := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "k6.md"}}
	agent := agentWithRefs([]aiv1.AIAgentSkill{
		{Name: "notes", Description: "Notes.", Content: "Some notes."},
		{Name: "k6-load-testing", SkillFrom: from, AlwaysInclude: true},
	}, nil)
	require.NoError(t, ResolveSkillSources(agent, cms.read))
	assert.Equal(t, aiv1.AIAgentSkill{
		Name:          "k6-load-testing",
		Description:   "Use when writing or reviewing a k6 load test.",
		Content:       "# k6 load testing\n\nStart from a smoke test.\n",
		AlwaysInclude: true,
		Metadata:      map[string]string{"author": "qa-team", "version": "1.2"},
	}, agent.Spec.Skills[1])
	assert.Equal(t, "Notes.", agent.Spec.Skills[0].Description, "an inline Skill is left alone")
	require.NoError(t, Normalize(agent))

	wrong := agentWithRefs([]aiv1.AIAgentSkill{{Name: "triage", SkillFrom: from}}, nil)
	err := ResolveSkillSources(wrong, cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	assert.Contains(t, err.Error(), "spec.skills[0].skillFrom", "the SKILL.md names k6-load-testing")
}

// A field written beside skillFrom is refused before the SKILL.md is read,
// as Normalize refuses it, rather than replaced by the file's value.
func TestResolveSkillSources_InlineFieldsBesideSkillFrom(t *testing.T) {
	cms := &configMaps{data: map[string]map[string]string{"skills": {"SKILL.md": k6SkillMarkdown}}}
	from := &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: "skills", Key: "SKILL.md"}}

	for _, key := range []string{"description", "license", "compatibility", "metadata", "allowed-tools", "content"} {
		spec := aiv1.SkillSpec{SkillFrom: from}
		switch key {
		case "description":
			spec.Description = "Mine."
		case "license":
			spec.License = "MIT"
		case "compatibility":
			spec.Compatibility = "Any."
		case "metadata":
			spec.Metadata = map[string]string{"a": "b"}
		case "allowed-tools":
			spec.AllowedTools = "Read"
		case "content":
			spec.Content = "Mine."
		}
		s := newSkill("k6-load-testing", spec)
		err := ResolveSkillSources(s, cms.read)
		require.ErrorIs(t, err, ErrInvalidSkillSource, key)
		assert.Contains(t, err.Error(), "spec."+key+": Forbidden: comes from the SKILL.md that skillFrom reads", key)
		assert.Equal(t, from, s.Spec.SkillFrom, key)
	}
	assert.Empty(t, cms.asked, "nothing is read")

	agent := agentWithRefs([]aiv1.AIAgentSkill{
		{Name: "k6-load-testing", SkillFrom: from, Description: "Mine.", Content: "Mine.", Metadata: map[string]string{"a": "b"}},
	}, nil)
	err := ResolveSkillSources(agent, cms.read)
	require.ErrorIs(t, err, ErrInvalidSkillSource)
	for _, key := range []string{"description", "content", "metadata"} {
		assert.Contains(t, err.Error(), "spec.skills[0]."+key+": Forbidden: comes from the SKILL.md that skillFrom reads", key)
	}
	assert.Equal(t, "Mine.", agent.Spec.Skills[0].Description)
	assert.Empty(t, cms.asked, "nothing is read")
}

func TestSkillSourceConfigMaps(t *testing.T) {
	from := func(name string) *aiv1.SkillSource {
		return &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: name, Key: "SKILL.md"}}
	}
	agent := agentWithRefs([]aiv1.AIAgentSkill{
		{Name: "a", SkillFrom: from("team-skills")},
		{Name: "b", Description: "B.", Content: "B."},
		{Name: "c", SkillFrom: from("base-skills")},
		{Name: "d", SkillFrom: from("team-skills")},
	}, nil)
	assert.Equal(t, []string{"base-skills", "team-skills"}, SkillSourceConfigMaps(agent))
	assert.Equal(t, []string{"skills"}, SkillSourceConfigMaps(newSkill("k6", aiv1.SkillSpec{SkillFrom: from("skills")})))
	assert.Empty(t, SkillSourceConfigMaps(newSkill("k6", aiv1.SkillSpec{Content: "C."})))
	assert.Empty(t, SkillSourceConfigMaps(&aiv1.MCPServer{}))
}

// A Skill is an environment kind of the AI group, applied before the agents
// that refer to it, and its status is never sent.
func TestSkillKind(t *testing.T) {
	kind, ok := LookupName("skills")
	require.True(t, ok)
	assert.Equal(t, aiv1.SkillKind, kind.Kind)
	assert.Equal(t, ScopeEnvironment, kind.Scope)
	looked, err := LookupKind("ai.testkube.io/v1alpha1", "Skill")
	require.NoError(t, err)
	assert.Equal(t, kind.Resource, looked.Resource)

	objects, err := decodeFile(t, "valid/skill.yaml")
	require.NoError(t, err)
	objects = append([]Object{objects[2]}, objects[:2]...)
	SortForApply(objects)
	var order []string
	for _, obj := range objects {
		k, err := KindOf(obj)
		require.NoError(t, err)
		order = append(order, k.Kind+"/"+obj.GetName())
	}
	assert.Equal(t, []string{"Skill/k6-load-testing", "Skill/release-notes", "AIAgent/load-tester"}, order)

	s := newSkill("k6", aiv1.SkillSpec{Description: "D.", Content: "C."})
	s.Status = commonv1.ControlPlaneResourceStatus{ID: "abc"}
	s.ResourceVersion = "3"
	prepared, err := PrepareForApply(s)
	require.NoError(t, err)
	assert.Empty(t, prepared.(*aiv1.Skill).Status)
	assert.Empty(t, prepared.GetResourceVersion())
	assert.Equal(t, "abc", StatusOf(s).ID)

	payload, err := ApplyPayload(s)
	require.NoError(t, err)
	assert.JSONEq(t, `{"apiVersion":"ai.testkube.io/v1alpha1","kind":"Skill","metadata":{"name":"k6"},"spec":{"description":"D.","content":"C."}}`, string(payload))
}
